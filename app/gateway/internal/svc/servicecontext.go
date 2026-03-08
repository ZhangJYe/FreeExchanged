package svc

import (
	articleClient "freeexchanged/app/article/cmd/rpc/articleclient"
	"freeexchanged/app/gateway/internal/config"
	"freeexchanged/app/gateway/internal/middleware"
	interactionclient "freeexchanged/app/interaction/cmd/rpc/interaction"
	rankingclient "freeexchanged/app/ranking/cmd/rpc/rankingclient"
	rateclient "freeexchanged/app/rate/cmd/rpc/rateclient"
	"freeexchanged/app/user/cmd/rpc/userclient"
	watchlistClient "freeexchanged/app/watchlist/cmd/rpc/watchlistClient"
	"freeexchanged/pkg/token"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config              config.Config
	PasetoMiddleware    rest.Middleware
	RateLimitMiddleware rest.Middleware
	UserRpc             userclient.User
	InteractionRpc      interactionclient.Interaction
	RankingRpc          rankingclient.Ranking
	RateRpc             rateclient.Rate
	ArticleRpc          articleClient.Article
	WatchlistRpc        watchlistClient.Watchlist
	BizRedis            *redis.Redis
	TokenMaker          token.Maker
}

func NewServiceContext(c config.Config) *ServiceContext {
	rds := redis.New(c.Redis.Host)
	maker, err := token.NewPasetoMakerFromBase64Key(c.Identity.AccessSecret, "", "")
	if err != nil {
		panic(err)
	}

	return &ServiceContext{
		Config:              c,
		PasetoMiddleware:    middleware.NewPasetoMiddleware(maker, rds).Handle,
		RateLimitMiddleware: middleware.NewRateLimitMiddleware(10, 100).Handle, // 限制10QPS，突发100
		UserRpc:             userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		InteractionRpc:      interactionclient.NewInteraction(zrpc.MustNewClient(c.InteractionRpc)),
		RankingRpc:          rankingclient.NewRanking(zrpc.MustNewClient(c.RankingRpc)),
		RateRpc:             rateclient.NewRate(zrpc.MustNewClient(c.RateRpc)),
		ArticleRpc:          articleClient.NewArticle(zrpc.MustNewClient(c.ArticleRpc)),
		WatchlistRpc:        watchlistClient.NewWatchlist(zrpc.MustNewClient(c.WatchlistRpc)),
		BizRedis:            rds,
		TokenMaker:          maker,
	}
}
