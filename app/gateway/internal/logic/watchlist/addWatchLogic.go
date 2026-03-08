package watchlist

import (
	"context"
	"encoding/json"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	watchlistClient "freeexchanged/app/watchlist/cmd/rpc/watchlistClient"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddWatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddWatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddWatchLogic {
	return &AddWatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddWatchLogic) AddWatch(req *types.AddWatchReq) (resp *types.AddWatchResp, err error) {
	userId := extractUserId(l.ctx)

	_, err = l.svcCtx.WatchlistRpc.AddWatch(l.ctx, &watchlistClient.AddWatchReq{
		UserId:       userId,
		CurrencyPair: req.CurrencyPair,
	})
	if err != nil {
		return nil, err
	}
	return &types.AddWatchResp{}, nil
}

func extractUserId(ctx context.Context) int64 {
	v := ctx.Value("userId")
	if v == nil {
		return 0
	}
	switch id := v.(type) {
	case int64:
		return id
	case json.Number:
		n, _ := id.Int64()
		return n
	case float64:
		return int64(id)
	}
	return 0
}
