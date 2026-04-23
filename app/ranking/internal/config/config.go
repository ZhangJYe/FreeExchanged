package config

import "github.com/zeromicro/go-zero/core/stores/redis"

type Config struct {
	BizRedis redis.RedisConf
	Kafka    struct {
		Brokers []string
	}
}
