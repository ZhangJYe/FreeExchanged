package config

import "github.com/zeromicro/go-zero/core/stores/redis"

type Config struct {
	DataSource string
	BizRedis   redis.RedisConf
}
