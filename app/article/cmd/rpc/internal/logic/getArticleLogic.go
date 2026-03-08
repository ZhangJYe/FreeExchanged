package logic

import (
	"context"

	"freeexchanged/app/article/cmd/rpc/article"
	"freeexchanged/app/article/cmd/rpc/internal/svc"
	"freeexchanged/app/article/model"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetArticleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetArticleLogic {
	return &GetArticleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetArticleLogic) GetArticle(in *article.GetArticleReq) (*article.GetArticleResp, error) {
	a, err := l.svcCtx.ArticleModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "Article not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &article.GetArticleResp{
		Article: &article.ArticleInfo{
			Id:         a.Id,
			Title:      a.Title,
			Content:    a.Content,
			AuthorId:   a.AuthorId,
			Status:     a.Status,
			LikeCount:  a.LikeCount,
			ViewCount:  a.ViewCount,
			CreateTime: a.CreateTime.Unix(),
			UpdateTime: a.UpdateTime.Unix(),
		},
	}, nil
}
