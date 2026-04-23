package ws

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"freeexchanged/app/gateway/internal/middleware"
	"freeexchanged/app/gateway/internal/svc"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

func HandleWS(svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) {
	tokenStr := tokenFromRequest(r)
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	payload, err := svcCtx.TokenMaker.VerifyToken(tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	exists, err := svcCtx.BizRedis.Exists(fmt.Sprintf("%s%s", middleware.TokenBlacklistPrefix, payload.ID))
	if err != nil {
		logx.Errorf("[WS] check token blacklist failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "token revoked", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, wsResponseHeader(r))
	if err != nil {
		logx.Errorf("[WS] upgrade failed: %v", err)
		return
	}

	userId := payload.UserID
	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	Hub.Add(userId, conn)
	go readLoop(userId, conn)
}

func tokenFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	parts := strings.Fields(authHeader)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}

	protocols := websocket.Subprotocols(r)
	for i, protocol := range protocols {
		if protocol == "auth" && i+1 < len(protocols) {
			return protocols[i+1]
		}
	}

	return r.URL.Query().Get("token")
}

func wsResponseHeader(r *http.Request) http.Header {
	for _, protocol := range websocket.Subprotocols(r) {
		if protocol == "auth" {
			return http.Header{"Sec-WebSocket-Protocol": []string{"auth"}}
		}
	}
	return nil
}

func readLoop(userId int64, conn *websocket.Conn) {
	defer Hub.Remove(userId, conn)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
