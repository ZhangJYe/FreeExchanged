package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	DataSource string
	BizRedis   redis.RedisConf
	Identity   struct {
		AccessSecret string
		AccessExpire int64
	}
}
