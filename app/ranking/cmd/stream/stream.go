package main

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"syscall"

	"freeexchanged/app/ranking/internal/config"
	"freeexchanged/app/ranking/internal/stream"
	"freeexchanged/pkg/metricsserver"

	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/stream.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	metricsserver.Start(ctx, c.Prometheus, "ranking-stream")

	consumer := stream.NewConsumer(ctx, c)
	defer consumer.Close()
	consumer.Start()

	fmt.Println("Starting ranking stream worker...")
	<-ctx.Done()
}
