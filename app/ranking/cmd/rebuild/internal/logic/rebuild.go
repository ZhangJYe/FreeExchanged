package logic

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"freeexchanged/app/ranking/cmd/rebuild/internal/config"
	"freeexchanged/app/ranking/internal/constant"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const rankingSnapshotQuery = `
SELECT
  a.id AS article_id,
  COALESCE(UNIX_TIMESTAMP(a.create_time), 0) AS published_score,
  COALESCE(s.like_users, 0) AS like_users,
  COALESCE(s.read_count, 0) AS read_count
FROM articles a
LEFT JOIN (
  SELECT
    article_id,
    SUM(CASE WHEN liked = 1 THEN 1 ELSE 0 END) AS like_users,
    COALESCE(SUM(read_count), 0) AS read_count
  FROM interaction_states
  GROUP BY article_id
) s ON s.article_id = a.id
WHERE a.status = 1
ORDER BY a.id`

type rankingSnapshotRow struct {
	ArticleID      int64 `db:"article_id"`
	PublishedScore int64 `db:"published_score"`
	LikeUsers      int64 `db:"like_users"`
	ReadCount      int64 `db:"read_count"`
}

func Run(ctx context.Context, c config.Config) error {
	start := time.Now()

	conn := sqlx.NewMysql(c.DataSource)
	rds := redis.MustNewRedis(c.BizRedis)

	var rows []rankingSnapshotRow
	if err := conn.QueryRowsCtx(ctx, &rows, rankingSnapshotQuery); err != nil {
		return fmt.Errorf("load ranking snapshot: %w", err)
	}

	if _, err := rds.Del(constant.RankingHotKey); err != nil {
		return fmt.Errorf("clear ranking redis key: %w", err)
	}

	for _, row := range rows {
		score := row.PublishedScore + row.LikeUsers*constant.RankingLikeScore + row.ReadCount*constant.RankingReadScore
		if _, err := rds.ZaddCtx(ctx, constant.RankingHotKey, score, strconv.FormatInt(row.ArticleID, 10)); err != nil {
			return fmt.Errorf("rebuild article %d score: %w", row.ArticleID, err)
		}
	}

	logx.Infof("ranking rebuild completed: articles=%d duration=%s", len(rows), time.Since(start))
	return nil
}
