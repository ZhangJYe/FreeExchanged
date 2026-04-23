package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	Identity struct {
		AccessSecret string
		AccessExpire int64
	}
	RateLimit struct {
		LikeLimit         int `json:",default=10"`
		LikeWindowSeconds int `json:",default=1"`
	}
	Redis          redis.RedisConf
	UserRpc        zrpc.RpcClientConf
	InteractionRpc zrpc.RpcClientConf
	RankingRpc     zrpc.RpcClientConf
	RateRpc        zrpc.RpcClientConf
	ArticleRpc     zrpc.RpcClientConf
	WatchlistRpc   zrpc.RpcClientConf
}
