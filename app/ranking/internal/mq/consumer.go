package main

import (
	"flag"
	"fmt"

	"freeexchanged/app/ranking/internal/config"

	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/mq.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Setup consumers
	// q := kq.MustNewQueue(c.KqConf, logic.NewConsumer(c))
	// defer q.Stop()
	// q.Start()
	fmt.Println("Ranking consumer started")
}
