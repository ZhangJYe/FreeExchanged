package logic

import (
	"context"

	"freeexchanged/app/article/cmd/rpc/article"
	"freeexchanged/app/article/cmd/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
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
	// todo: add your logic here and delete this line

	return &article.ListArticlesResp{}, nil
}
