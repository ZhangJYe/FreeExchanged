package model

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ WatchlistModel = (*customWatchlistModel)(nil)

type (
	WatchlistModel interface {
		watchlistModel
		withSession(session sqlx.Session) WatchlistModel
		FindAllByUserId(ctx context.Context, userId int64) ([]*Watchlist, error)
		DeleteByUserIdAndPair(ctx context.Context, userId int64, pair string) error
	}

	customWatchlistModel struct {
		*defaultWatchlistModel
	}
)

func NewWatchlistModel(conn sqlx.SqlConn) WatchlistModel {
	return &customWatchlistModel{
		defaultWatchlistModel: newWatchlistModel(conn),
	}
}

func (m *customWatchlistModel) withSession(session sqlx.Session) WatchlistModel {
	return NewWatchlistModel(sqlx.NewSqlConnFromSession(session))
}

// FindAllByUserId 查询用户全部自选记录（缓存 miss 时 fallback 用）
func (m *customWatchlistModel) FindAllByUserId(ctx context.Context, userId int64) ([]*Watchlist, error) {
	query := "SELECT " + watchlistRows + " FROM `watchlist` WHERE `user_id` = ? ORDER BY `create_time` DESC"
	var list []*Watchlist
	err := m.conn.QueryRowsCtx(ctx, &list, query, userId)
	return list, err
}

// DeleteByUserIdAndPair 按 userId + currency_pair 删除
func (m *customWatchlistModel) DeleteByUserIdAndPair(ctx context.Context, userId int64, pair string) error {
	query := "DELETE FROM `watchlist` WHERE `user_id` = ? AND `currency_pair` = ?"
	_, err := m.conn.ExecCtx(ctx, query, userId, pair)
	return err
}
