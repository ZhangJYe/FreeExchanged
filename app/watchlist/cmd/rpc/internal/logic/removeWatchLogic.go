package logic

import (
	"context"
	"fmt"

	"freeexchanged/app/watchlist/cmd/rpc/internal/svc"
	"freeexchanged/app/watchlist/cmd/rpc/watchlist"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RemoveWatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoveWatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveWatchLogic {
	return &RemoveWatchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RemoveWatchLogic) RemoveWatch(in *watchlist.RemoveWatchReq) (*watchlist.RemoveWatchResp, error) {
	if in.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user is required")
	}

	pair, ok := normalizeCurrencyPair(in.CurrencyPair)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "currency_pair must be BASE/QUOTE")
	}

	if err := l.svcCtx.WatchlistModel.DeleteByUserIdAndPair(l.ctx, in.UserId, pair); err != nil {
		logx.Errorf("[Watchlist] RemoveWatch DB err: %v", err)
		return nil, status.Error(codes.Internal, "db delete failed")
	}

	key := fmt.Sprintf("watchlist:%d", in.UserId)
	if _, err := l.svcCtx.Redis.Srem(key, pair); err != nil {
		logx.Errorf("[Watchlist] Redis SREM err: %v", err)
	}

	logx.Infof("[Watchlist] user %d removed %s", in.UserId, pair)
	return &watchlist.RemoveWatchResp{}, nil
}
