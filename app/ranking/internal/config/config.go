package config

import (
	"freeexchanged/pkg/metricsserver"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	BizRedis   redis.RedisConf
	Prometheus metricsserver.Config
	Kafka      struct {
		Brokers []string
	}
}
