package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var fixedWindowLimiterScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[2])
end
local ttl = redis.call("TTL", KEYS[1])
if current > tonumber(ARGV[1]) then
  return {0, ttl}
end
return {1, ttl}
`)

type RateLimitMiddleware struct {
	redis         *redis.Redis
	limit         int
	windowSeconds int
}

func NewRateLimitMiddleware(rds *redis.Redis, limit, windowSeconds int) *RateLimitMiddleware {
	if limit <= 0 {
		limit = 10
	}
	if windowSeconds <= 0 {
		windowSeconds = 1
	}

	return &RateLimitMiddleware{
		redis:         rds,
		limit:         limit,
		windowSeconds: windowSeconds,
	}
}

func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := m.key(r)
		allowed, retryAfter, err := m.allow(r.Context(), key)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("RateLimit: redis check failed, key=%s, err=%v", key, err)
			next(w, r)
			return
		}
		if !allowed {
			logx.WithContext(r.Context()).Errorf("RateLimit: rejected, key=%s, url=%s", key, r.URL.Path)
			writeRateLimited(w, retryAfter)
			return
		}
		next(w, r)
	}
}

func (m *RateLimitMiddleware) allow(ctx context.Context, key string) (bool, int, error) {
	if m == nil || m.redis == nil {
		return true, 0, nil
	}

	result, err := m.redis.ScriptRunCtx(ctx, fixedWindowLimiterScript, []string{key}, m.limit, m.windowSeconds)
	if err != nil {
		return false, 0, err
	}

	allowed, retryAfter, err := parseLimiterResult(result)
	if retryAfter <= 0 {
		retryAfter = m.windowSeconds
	}
	return allowed, retryAfter, err
}

func (m *RateLimitMiddleware) key(r *http.Request) string {
	if userID, ok := r.Context().Value("userId").(int64); ok && userID > 0 {
		return fmt.Sprintf("rate:like:user:%d", userID)
	}
	return "rate:like:ip:" + clientIP(r)
}

func parseLimiterResult(result any) (bool, int, error) {
	values, ok := result.([]any)
	if !ok || len(values) < 2 {
		return false, 0, fmt.Errorf("unexpected redis script result: %T", result)
	}

	allowed, err := toInt64(values[0])
	if err != nil {
		return false, 0, err
	}
	retryAfter, err := toInt64(values[1])
	if err != nil {
		return false, 0, err
	}
	return allowed == 1, int(retryAfter), nil
}

func toInt64(v any) (int64, error) {
	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	case string:
		return strconv.ParseInt(val, 10, 64)
	case []byte:
		return strconv.ParseInt(string(val), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected integer value: %T", v)
	}
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if ip := strings.TrimSpace(strings.Split(forwarded, ",")[0]); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeRateLimited(w http.ResponseWriter, retryAfter int) {
	if retryAfter <= 0 {
		retryAfter = int(time.Second.Seconds())
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"code":429,"msg":"too many requests"}`))
}
