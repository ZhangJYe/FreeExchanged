package main

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"syscall"

	"freeexchanged/app/interaction/internal/outbox"
	"freeexchanged/pkg/metricsserver"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/outbox.yaml", "the config file")

func main() {
	flag.Parse()

	var c outbox.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	metricsserver.Start(ctx, metricsserver.Config{
		Host: c.Prometheus.Host,
		Port: c.Prometheus.Port,
		Path: c.Prometheus.Path,
	}, "interaction-outbox")

	fmt.Println("Starting interaction outbox dispatcher...")
	outbox.NewDispatcher(c).Run(ctx)
}
