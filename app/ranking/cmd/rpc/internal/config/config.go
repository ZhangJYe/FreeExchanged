package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	BizRedis redis.RedisConf
	RabbitMQ struct {
		Host     string
		Port     int
		Username string
		Password string
		VHost    string
	}
}
