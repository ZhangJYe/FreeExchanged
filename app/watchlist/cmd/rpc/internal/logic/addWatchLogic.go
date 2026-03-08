package logic

import (
	"context"
	"fmt"

	"freeexchanged/app/watchlist/cmd/rpc/internal/svc"
	"freeexchanged/app/watchlist/cmd/rpc/watchlist"
	"freeexchanged/app/watchlist/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddWatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddWatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddWatchLogic {
	return &AddWatchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AddWatchLogic) AddWatch(in *watchlist.AddWatchReq) (*watchlist.AddWatchResp, error) {
	// 1. 写 MySQL（UNIQUE KEY 保证幂等，重复插入忽略）
	_, err := l.svcCtx.WatchlistModel.Insert(l.ctx, &model.Watchlist{
		UserId:       in.UserId,
		CurrencyPair: in.CurrencyPair,
	})
	if err != nil {
		// 重复键不报错（mysql: Error 1062），其他错误打日志但不阻断
		logx.Errorf("[Watchlist] AddWatch insert err (may be duplicate): %v", err)
	}

	// 2. Write-Through: 同步写 Redis Set
	key := fmt.Sprintf("watchlist:%d", in.UserId)
	if _, err := l.svcCtx.Redis.Sadd(key, in.CurrencyPair); err != nil {
		logx.Errorf("[Watchlist] Redis SADD err: %v", err)
		// Redis 失败不影响业务
	}

	logx.Infof("[Watchlist] user %d added %s", in.UserId, in.CurrencyPair)
	return &watchlist.AddWatchResp{}, nil
}
