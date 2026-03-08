package svc

import (
	rateClient "freeexchanged/app/rate/cmd/rpc/rateclient"
	"freeexchanged/app/watchlist/cmd/rpc/internal/config"
	"freeexchanged/app/watchlist/model"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config         config.Config
	WatchlistModel model.WatchlistModel
	Redis          *redis.Redis
	RateRpc        rateClient.Rate
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:         c,
		WatchlistModel: model.NewWatchlistModel(sqlx.NewMysql(c.DataSource)),
		Redis:          redis.New(c.BizRedis.Host),
		RateRpc:        rateClient.NewRate(zrpc.MustNewClient(c.RateRpc)),
	}
}
