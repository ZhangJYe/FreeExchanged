package main

import (
	"flag"
	"fmt"

	"freeexchanged/app/interaction/cmd/rpc/internal/config"
	"freeexchanged/app/interaction/cmd/rpc/internal/server"
	"freeexchanged/app/interaction/cmd/rpc/internal/svc"
	"freeexchanged/app/interaction/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/consul"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/interaction.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterInteractionServer(grpcServer, server.NewInteractionServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	// 注册到 Consul
	if err := consul.RegisterService(c.ListenOn, c.Consul); err != nil {
		panic(err)
	}

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
