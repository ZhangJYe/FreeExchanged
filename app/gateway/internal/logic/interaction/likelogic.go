// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"
	"encoding/json"
	"errors"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	interactionclient "freeexchanged/app/interaction/cmd/rpc/interaction"

	"github.com/zeromicro/go-zero/core/logx"
)

type LikeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 点赞/取消点赞
func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LikeLogic) Like(req *types.LikeReq) (resp *types.LikeResp, err error) {
	var userId int64
	if v, ok := l.ctx.Value("userId").(json.Number); ok {
		userId, _ = v.Int64()
	} else if v, ok := l.ctx.Value("userId").(int64); ok {
		userId = v
	}
	if userId <= 0 {
		return nil, errors.New("missing user id")
	}

	switch req.Action {
	case 0, 1:
		_, err = l.svcCtx.InteractionRpc.Like(l.ctx, &interactionclient.LikeReq{
			UserId:    userId,
			ArticleId: req.ArticleId,
		})
	case 2:
		_, err = l.svcCtx.InteractionRpc.Unlike(l.ctx, &interactionclient.UnlikeReq{
			UserId:    userId,
			ArticleId: req.ArticleId,
		})
	default:
		return nil, errors.New("invalid action")
	}

	return &types.LikeResp{}, err
}
