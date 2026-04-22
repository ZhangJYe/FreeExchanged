package main

import (
	"flag"

	"freeexchanged/app/rate/cmd/job/internal/config"
	"freeexchanged/app/rate/cmd/job/internal/logic"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/job.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	logx.Info("Job started")

	// Start your job logic here
	logic.Run(c)
}
