package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"freeexchanged/pkg/token"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const TokenBlacklistPrefix = "token:blacklist:"

// PasetoMiddleware 鉴权中间件
type PasetoMiddleware struct {
	TokenMaker token.Maker
	Redis      *redis.Redis
}

func NewPasetoMiddleware(maker token.Maker, rds *redis.Redis) *PasetoMiddleware {
	return &PasetoMiddleware{
		TokenMaker: maker,
		Redis:      rds,
	}
}

// Handle 核心处理逻辑
func (m *PasetoMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 获取 Authorization Header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			httpx.Error(w, errors.New("missing authorization header"))
			return
		}

		// 2. 也是最重要的安全性步骤：提取 Token
		// 格式通常为 "Bearer <token>"
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			httpx.Error(w, errors.New("invalid authorization header format"))
			return
		}

		accessToken := parts[1]

		// 3. 验证 Token
		payload, err := m.TokenMaker.VerifyToken(accessToken)
		if err != nil {
			logx.Errorf("Verify token failed: %v", err)
			// 返回 401 Unauthorized
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		// 4. [新增] 检查黑名单 (Redis)
		key := fmt.Sprintf("%s%s", TokenBlacklistPrefix, payload.ID)
		exists, err := m.Redis.Exists(key)
		if err != nil {
			logx.Errorf("Check token blacklist failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if exists {
			logx.Infof("Token is in blacklist: %s", payload.ID)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Token Revoked"))
			return
		}

		// 5. 将 UserId 注入 Context，供后续 Logic 使用
		// key 建议统一封装个常量，这里演示直接用字符串
		ctx := context.WithValue(r.Context(), "userId", payload.UserID)
		// 同时把 token 也塞进去，方便 logout 时使用
		ctx = context.WithValue(ctx, "token", accessToken)

		next(w, r.WithContext(ctx))
	}
}
