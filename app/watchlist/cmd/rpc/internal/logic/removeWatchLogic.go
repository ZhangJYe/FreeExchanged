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
	// 1. 删除 MySQL
	if err := l.svcCtx.WatchlistModel.DeleteByUserIdAndPair(l.ctx, in.UserId, in.CurrencyPair); err != nil {
		logx.Errorf("[Watchlist] RemoveWatch DB err: %v", err)
		return nil, status.Error(codes.Internal, "db delete failed")
	}

	// 2. Write-Through: 同步删除 Redis Set 成员
	key := fmt.Sprintf("watchlist:%d", in.UserId)
	if _, err := l.svcCtx.Redis.Srem(key, in.CurrencyPair); err != nil {
		logx.Errorf("[Watchlist] Redis SREM err: %v", err)
	}

	logx.Infof("[Watchlist] user %d removed %s", in.UserId, in.CurrencyPair)
	return &watchlist.RemoveWatchResp{}, nil
}
