package watchlist

import (
	"context"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	watchlistClient "freeexchanged/app/watchlist/cmd/rpc/watchlistClient"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetWatchlistLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetWatchlistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWatchlistLogic {
	return &GetWatchlistLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetWatchlistLogic) GetWatchlist() (resp *types.GetWatchlistResp, err error) {
	userId := extractUserId(l.ctx)

	rpcResp, err := l.svcCtx.WatchlistRpc.GetWatchlist(l.ctx, &watchlistClient.GetWatchlistReq{
		UserId: userId,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.WatchItem, len(rpcResp.Items))
	for i, item := range rpcResp.Items {
		items[i] = types.WatchItem{
			CurrencyPair: item.CurrencyPair,
			Rate:         item.Rate,
			UpdatedAt:    item.UpdatedAt,
		}
	}
	return &types.GetWatchlistResp{Items: items}, nil
}
