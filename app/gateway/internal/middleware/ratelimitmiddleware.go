package middleware

import (
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/time/rate"
)

type RateLimitMiddleware struct {
	limiter *rate.Limiter
}

func NewRateLimitMiddleware(r rate.Limit, b int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: rate.NewLimiter(r, b),
	}
}

func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.limiter.Allow() {
			// 建议：不要每次都 Errorf，限流属于“预期行为”
			logx.WithContext(r.Context()).Errorf("RateLimit: rejected, url=%s", r.URL.Path)

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			// 这个可选：告诉客户端多久后再试（你也可以按配置计算更合理值）
			w.Header().Set("Retry-After", time.Second.String())

			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"code":429,"msg":"too many requests"}`))
			return
		}
		next(w, r)
	}
}
