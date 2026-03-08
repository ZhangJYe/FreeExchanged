package config

import (
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/consul"
)

type Config struct {
	zrpc.RpcServerConf
	Consul   consul.Conf
	RabbitMQ struct {
		Host     string
		Port     int
		Username string
		Password string
		VHost    string
	}
}
