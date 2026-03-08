package watchlist

import (
	"context"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	watchlistClient "freeexchanged/app/watchlist/cmd/rpc/watchlistClient"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveWatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveWatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveWatchLogic {
	return &RemoveWatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveWatchLogic) RemoveWatch(req *types.RemoveWatchReq) (resp *types.RemoveWatchResp, err error) {
	userId := extractUserId(l.ctx)

	_, err = l.svcCtx.WatchlistRpc.RemoveWatch(l.ctx, &watchlistClient.RemoveWatchReq{
		UserId:       userId,
		CurrencyPair: req.CurrencyPair,
	})
	if err != nil {
		return nil, err
	}
	return &types.RemoveWatchResp{}, nil
}
