package main

import (
	"flag"
	"fmt"

	"freeexchanged/app/rate/cmd/rpc/internal/config"
	"freeexchanged/app/rate/cmd/rpc/internal/server"
	"freeexchanged/app/rate/cmd/rpc/internal/svc"
	"freeexchanged/app/rate/cmd/rpc/rate"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/rate.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		rate.RegisterRateServer(grpcServer, server.NewRateServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
