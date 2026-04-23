package logic

import (
	"context"

	"freeexchanged/app/article/cmd/rpc/article"
	"freeexchanged/app/article/cmd/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ListArticlesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListArticlesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListArticlesLogic {
	return &ListArticlesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListArticlesLogic) ListArticles(in *article.ListArticlesReq) (*article.ListArticlesResp, error) {
	rows, err := l.svcCtx.ArticleModel.FindPage(l.ctx, in.AuthorId, in.Page, in.PageSize)
	if err != nil {
		l.Logger.Errorf("ListArticles query failed: %v", err)
		return nil, status.Error(codes.Internal, "list articles failed")
	}

	items := make([]*article.ArticleInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, &article.ArticleInfo{
			Id:         row.Id,
			Title:      row.Title,
			Content:    row.Content,
			AuthorId:   row.AuthorId,
			Status:     row.Status,
			LikeCount:  row.LikeCount,
			ViewCount:  row.ViewCount,
			CreateTime: row.CreateTime.Unix(),
			UpdateTime: row.UpdateTime.Unix(),
		})
	}

	return &article.ListArticlesResp{Articles: items}, nil
}
