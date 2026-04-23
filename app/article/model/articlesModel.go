package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ArticlesModel = (*customArticlesModel)(nil)

type (
	// ArticlesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customArticlesModel.
	ArticlesModel interface {
		articlesModel
		FindPage(ctx context.Context, authorID int64, page, pageSize int32) ([]*Articles, error)
		withSession(session sqlx.Session) ArticlesModel
	}

	customArticlesModel struct {
		*defaultArticlesModel
	}
)

// NewArticlesModel returns a model for the database table.
func NewArticlesModel(conn sqlx.SqlConn) ArticlesModel {
	return &customArticlesModel{
		defaultArticlesModel: newArticlesModel(conn),
	}
}

func (m *customArticlesModel) withSession(session sqlx.Session) ArticlesModel {
	return NewArticlesModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customArticlesModel) FindPage(ctx context.Context, authorID int64, page, pageSize int32) ([]*Articles, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := int64(page-1) * int64(pageSize)
	args := []any{int64(pageSize), offset}
	where := "where `status` = 1"
	if authorID > 0 {
		where += " and `author_id` = ?"
		args = []any{authorID, int64(pageSize), offset}
	}

	query := fmt.Sprintf("select %s from %s %s order by `id` desc limit ? offset ?", articlesRows, m.table, where)
	var resp []*Articles
	if err := m.conn.QueryRowsCtx(ctx, &resp, query, args...); err != nil {
		return nil, err
	}
	return resp, nil
}
