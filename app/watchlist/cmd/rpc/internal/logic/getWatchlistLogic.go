package logic

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	rateClient "freeexchanged/app/rate/cmd/rpc/rateclient"
	"freeexchanged/app/watchlist/cmd/rpc/internal/svc"
	"freeexchanged/app/watchlist/cmd/rpc/watchlist"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxRateFanoutConcurrency = 8
	rateRPCTimeout           = 2 * time.Second
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
	if in.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user is required")
	}

	key := fmt.Sprintf("watchlist:%d", in.UserId)
	pairs, err := l.svcCtx.Redis.Smembers(key)
	if err != nil || len(pairs) == 0 {
		if err != nil {
			logx.Errorf("[Watchlist] cache read failed, fallback to db, user=%d, err=%v", in.UserId, err)
		} else {
			logx.Infof("[Watchlist] cache miss, fallback to db, user=%d", in.UserId)
		}

		records, dbErr := l.svcCtx.WatchlistModel.FindAllByUserId(l.ctx, in.UserId)
		if dbErr != nil {
			logx.Errorf("[Watchlist] db query failed: %v", dbErr)
			return &watchlist.GetWatchlistResp{}, nil
		}

		pairs = pairs[:0]
		cacheValues := make([]any, 0, len(records))
		for _, r := range records {
			pair, ok := normalizeCurrencyPair(r.CurrencyPair)
			if !ok {
				logx.Errorf("[Watchlist] skip invalid stored pair user=%d pair=%q", in.UserId, r.CurrencyPair)
				continue
			}
			pairs = append(pairs, pair)
			cacheValues = append(cacheValues, pair)
		}
		if len(cacheValues) > 0 {
			if _, err := l.svcCtx.Redis.Sadd(key, cacheValues...); err != nil {
				logx.Errorf("[Watchlist] cache backfill failed user=%d err=%v", in.UserId, err)
			}
		}
	}

	pairs = normalizePairList(pairs)
	if len(pairs) == 0 {
		return &watchlist.GetWatchlistResp{}, nil
	}

	type result struct {
		index int
		item  *watchlist.WatchItem
	}

	workerCount := len(pairs)
	if workerCount > maxRateFanoutConcurrency {
		workerCount = maxRateFanoutConcurrency
	}

	jobs := make(chan int)
	results := make(chan result, len(pairs))

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				pair := pairs[idx]
				base, target := parsePair(pair)
				item := &watchlist.WatchItem{CurrencyPair: pair}

				callCtx, cancel := context.WithTimeout(l.ctx, rateRPCTimeout)
				rateResp, err := l.svcCtx.RateRpc.GetRate(callCtx, &rateClient.GetRateReq{
					From: base,
					To:   target,
				})
				cancel()

				if err != nil {
					logx.Errorf("[Watchlist] GetRate failed pair=%s err=%v", pair, err)
				} else {
					item.Rate = rateResp.GetRate()
					item.UpdatedAt = rateResp.GetUpdatedAt()
				}
				results <- result{index: idx, item: item}
			}
		}()
	}

	go func() {
		for idx := range pairs {
			jobs <- idx
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	items := make([]*watchlist.WatchItem, len(pairs))
	for r := range results {
		items[r.index] = r.item
	}

	return &watchlist.GetWatchlistResp{Items: items}, nil
}

func parsePair(pair string) (string, string) {
	normalized, ok := normalizeCurrencyPair(pair)
	if !ok {
		return pair, ""
	}

	parts := strings.Split(normalized, "/")
	return parts[0], parts[1]
}

func normalizeCurrencyPair(pair string) (string, bool) {
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(pair)), "/")
	if len(parts) != 2 {
		return "", false
	}

	base := strings.TrimSpace(parts[0])
	target := strings.TrimSpace(parts[1])
	if base == "" || target == "" || base == target {
		return "", false
	}

	return base + "/" + target, true
}

func normalizePairList(pairs []string) []string {
	seen := make(map[string]struct{}, len(pairs))
	normalized := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		pair, ok := normalizeCurrencyPair(pair)
		if !ok {
			continue
		}
		if _, exists := seen[pair]; exists {
			continue
		}
		seen[pair] = struct{}{}
		normalized = append(normalized, pair)
	}

	sort.Strings(normalized)
	return normalized
}
