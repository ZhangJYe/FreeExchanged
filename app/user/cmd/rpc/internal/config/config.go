package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/consul" // 引入 consul 定义
)

type Config struct {
	zrpc.RpcServerConf
	DataSource string      // MySQL 配置
	Consul     consul.Conf // Consul 配置
	// 业务 Redis 配置 (BizRedis)
	// 不要叫 Redis，避免与 zrpc.RpcServerConf 里的 Redis 冲突
	BizRedis redis.RedisConf
	Identity struct { // Paseto/JWT 密钥 (改名 Identity 避开 Auth 冲突)
		AccessSecret string
		AccessExpire int64
	}
}
