package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"freeexchanged/app/ranking/cmd/rebuild/internal/config"
	"freeexchanged/app/ranking/cmd/rebuild/internal/logic"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/rebuild.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := logic.Run(ctx, c); err != nil {
		logx.Errorf("ranking rebuild failed: %v", err)
		os.Exit(1)
	}
}
