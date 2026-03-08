package logic

import (
	"context"
	"fmt"
	"strings"

	rateClient "freeexchanged/app/rate/cmd/rpc/rateclient"
	"freeexchanged/app/watchlist/cmd/rpc/internal/svc"
	"freeexchanged/app/watchlist/cmd/rpc/watchlist"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetWatchlistLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetWatchlistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWatchlistLogic {
	return &GetWatchlistLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetWatchlistLogic) GetWatchlist(in *watchlist.GetWatchlistReq) (*watchlist.GetWatchlistResp, error) {
	key := fmt.Sprintf("watchlist:%d", in.UserId)

	// ① 优先读 Redis Set（缓存命中）
	pairs, err := l.svcCtx.Redis.Smembers(key)
	if err != nil || len(pairs) == 0 {
		// ② Cache Miss：降级查 MySQL，并回填 Redis
		logx.Infof("[Watchlist] cache miss for user %d, fallback to DB", in.UserId)
		records, dbErr := l.svcCtx.WatchlistModel.FindAllByUserId(l.ctx, in.UserId)
		if dbErr != nil {
			logx.Errorf("[Watchlist] FindAllByUserId err: %v", dbErr)
			return &watchlist.GetWatchlistResp{Items: []*watchlist.WatchItem{}}, nil
		}
		for _, r := range records {
			pairs = append(pairs, r.CurrencyPair)
			l.svcCtx.Redis.Sadd(key, r.CurrencyPair) // 回填缓存
		}
	}

	if len(pairs) == 0 {
		return &watchlist.GetWatchlistResp{Items: []*watchlist.WatchItem{}}, nil
	}

	// ③ goroutine fan-out：并发调 Rate RPC 获取实时汇率
	type result struct {
		item *watchlist.WatchItem
	}
	ch := make(chan result, len(pairs))

	for _, pair := range pairs {
		go func(p string) {
			from, to := parsePair(p)
			rateResp, err := l.svcCtx.RateRpc.GetRate(l.ctx, &rateClient.GetRateReq{
				From: from,
				To:   to,
			})
			if err != nil {
				logx.Errorf("[Watchlist] GetRate for %s failed: %v", p, err)
				// 部分降级：返回汇率为 0，不影响其他货币对
				ch <- result{item: &watchlist.WatchItem{CurrencyPair: p, Rate: 0}}
				return
			}
			ch <- result{item: &watchlist.WatchItem{
				CurrencyPair: p,
				Rate:         rateResp.Rate,
				UpdatedAt:    rateResp.UpdatedAt,
			}}
		}(pair)
	}

	// ④ 收集所有 goroutine 结果
	items := make([]*watchlist.WatchItem, 0, len(pairs))
	for range pairs {
		r := <-ch
		items = append(items, r.item)
	}

	return &watchlist.GetWatchlistResp{Items: items}, nil
}

// parsePair 解析 "USD/CNY" → ("USD", "CNY")
func parsePair(pair string) (string, string) {
	parts := strings.SplitN(pair, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return pair, ""
}
