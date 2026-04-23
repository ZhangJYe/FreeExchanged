package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"freeexchanged/app/article/cmd/rpc/article"
	"freeexchanged/app/article/cmd/rpc/internal/svc"
	"freeexchanged/app/article/model"
	"freeexchanged/pkg/events"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PublishLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishLogic {
	return &PublishLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PublishLogic) Publish(in *article.PublishReq) (*article.PublishResp, error) {
	var articleID int64

	err := l.svcCtx.Conn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		articleModel := model.NewArticlesModel(sqlx.NewSqlConnFromSession(session))
		res, err := articleModel.Insert(ctx, &model.Articles{
			Title:    in.Title,
			Content:  in.Content,
			AuthorId: in.AuthorId,
			Status:   1,
		})
		if err != nil {
			return err
		}

		articleID, err = res.LastInsertId()
		if err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]any{
			"event_id":    uuid.NewString(),
			"event_type":  events.EventArticlePublished,
			"version":     1,
			"article_id":  articleID,
			"title":       in.Title,
			"author_id":   in.AuthorId,
			"occurred_at": time.Now().Unix(),
		})
		if err != nil {
			return err
		}

		_, err = session.ExecCtx(ctx, `
INSERT INTO article_outbox_events
  (aggregate_type, aggregate_id, event_type, topic, event_key, payload, status, next_retry_at)
VALUES
  (?, ?, ?, ?, ?, ?, 0, NOW())`,
			"article",
			articleID,
			events.EventArticlePublished,
			events.TopicArticleEvents,
			strconv.FormatInt(articleID, 10),
			string(payload),
		)
		return err
	})
	if err != nil {
		l.Logger.Errorf("Publish transaction failed: %v", err)
		return nil, status.Error(codes.Internal, "Publish Failed")
	}

	return &article.PublishResp{
		ArticleId: articleID,
	}, nil
}
