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
	// 1. 插入数据库
	newArticle := &model.Articles{
		Title:    in.Title,
		Content:  in.Content,
		AuthorId: in.AuthorId,
		Status:   1, // 默认直接发布
	}

	res, err := l.svcCtx.ArticleModel.Insert(l.ctx, newArticle)
	if err != nil {
		logx.Errorf("Publish DB Error: %v", err)
		return nil, status.Error(codes.Internal, "DB Insert Failed")
	}

	articleId, _ := res.LastInsertId()

	// 2. 发送 MQ 消息 (异步)
	// 如果 MQ 失败，业务流程上文章已发布成功，仅仅是没进排行榜，可以接受
	go func() {
		if err := l.svcCtx.Producer.PublishArticleEvent(articleId, in.Title, in.AuthorId); err != nil {
			logx.Errorf("Failed to send publish event for article %d: %v", articleId, err)
		}
	}()

	return &article.PublishResp{
		ArticleId: articleId,
	}, nil
}
