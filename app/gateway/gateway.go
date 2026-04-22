package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"freeexchanged/app/gateway/internal/config"
	"freeexchanged/app/gateway/internal/handler"
	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/ws"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// --- WebSocket 路由 ---
	// go-zero 的 rest.Server 支持直接添加自定义 http handler
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/ws",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ws.HandleWS(ctx, w, r)
		},
	})

	// --- 启动 Redis Pub/Sub 监听器 ---
	rdb := redis.NewClient(&redis.Options{
		Addr: c.Redis.Host, // e.g. "127.0.0.1:6380"
	})
	go ws.StartRedisListener(context.Background(), rdb)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
