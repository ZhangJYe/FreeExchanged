package logic

import (
	"context"
	"fmt"
	"strings"

	"freeexchanged/app/watchlist/cmd/rpc/internal/svc"
	"freeexchanged/app/watchlist/cmd/rpc/watchlist"
	"freeexchanged/app/watchlist/model"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	if in.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user is required")
	}

	pair, ok := normalizeCurrencyPair(in.CurrencyPair)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "currency_pair must be BASE/QUOTE")
	}

	_, err := l.svcCtx.WatchlistModel.Insert(l.ctx, &model.Watchlist{
		UserId:       in.UserId,
		CurrencyPair: pair,
	})
	if err != nil {
		if !strings.Contains(err.Error(), "Duplicate entry") {
			logx.Errorf("[Watchlist] AddWatch insert err: %v", err)
			return nil, status.Error(codes.Internal, "add watch failed")
		}
		logx.Infof("[Watchlist] duplicate add ignored user=%d pair=%s", in.UserId, pair)
	}

	key := fmt.Sprintf("watchlist:%d", in.UserId)
	if _, err := l.svcCtx.Redis.Sadd(key, pair); err != nil {
		logx.Errorf("[Watchlist] Redis SADD err: %v", err)
	}

	logx.Infof("[Watchlist] user %d added %s", in.UserId, pair)
	return &watchlist.AddWatchResp{}, nil
}
