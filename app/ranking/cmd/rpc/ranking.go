package main

import (
	"context" // Added import
	"flag"
	"fmt"

	"freeexchanged/app/ranking/cmd/rpc/internal/config"
	"freeexchanged/app/ranking/cmd/rpc/internal/mq"
	"freeexchanged/app/ranking/cmd/rpc/internal/server"
	"freeexchanged/app/ranking/cmd/rpc/internal/svc"
	"freeexchanged/app/ranking/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/ranking.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	// Start Kafka consumers for ranking events.
	if c.ConsumerEnabled {
		consumer := mq.NewArticleConsumer(context.Background(), ctx)
		if consumer != nil {
			consumer.Start()
			logx.Info("Kafka consumer started")
		}
	} else {
		logx.Info("Kafka consumer disabled by config")
	}

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterRankingServer(grpcServer, server.NewRankingServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
