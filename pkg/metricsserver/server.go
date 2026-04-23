package metricsserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zeromicro/go-zero/core/logx"
)

type Config struct {
	Host string `json:",default=0.0.0.0"`
	Port int    `json:",default=9100"`
	Path string `json:",default=/metrics"`
}

func Start(ctx context.Context, cfg Config, name string) {
	if cfg.Port <= 0 {
		return
	}

	path := cfg.Path
	if path == "" {
		path = "/metrics"
	}
	host := cfg.Host
	if host == "" {
		host = "0.0.0.0"
	}

	addr := net.JoinHostPort(host, strconv.Itoa(cfg.Port))
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	go func() {
		logx.Infof("%s metrics listening on %s%s", name, addr, path)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logx.Errorf("%s metrics server failed: %v", name, err)
		}
	}()
}
