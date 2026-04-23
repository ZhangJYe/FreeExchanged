package article

import (
	"context"
	"errors"

	articleClient "freeexchanged/app/article/cmd/rpc/articleclient"
	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishLogic {
	return &PublishLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublishLogic) Publish(req *types.PublishArticleReq) (resp *types.PublishArticleResp, err error) {
	// 从 Context 获取 userId (PasetoMiddleware 注入)
	var userId int64
	if v := l.ctx.Value("userId"); v != nil {
		if id, ok := v.(int64); ok {
			userId = id
		} else if idFloat, ok := v.(float64); ok {
			userId = int64(idFloat)
		}
	}
	if userId <= 0 {
		return nil, errors.New("missing user id")
	}

	res, err := l.svcCtx.ArticleRpc.Publish(l.ctx, &articleClient.PublishReq{
		Title:    req.Title,
		Content:  req.Content,
		AuthorId: userId,
	})
	if err != nil {
		return nil, err
	}

	return &types.PublishArticleResp{
		ArticleId: res.ArticleId,
	}, nil
}
