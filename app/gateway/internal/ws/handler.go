package ws

import (
	"freeexchanged/app/gateway/internal/svc"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

// HandleWS HTTP 升级 + PASETO 鉴权入口
// 路由: GET /ws?token=<paseto_token>
func HandleWS(svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) {
	// 1. 从 Query 取 token（浏览器 WS API 不支持自定义 Header）
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	// 2. 复用 TokenMaker 验证（纯计算，无 IO）
	payload, err := svcCtx.TokenMaker.VerifyToken(tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// 3. HTTP → WebSocket 协议升级
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Errorf("[WS] upgrade failed: %v", err)
		return
	}

	userId := payload.UserID

	// 4. 注册连接到 Hub
	Hub.Add(userId, conn)

	// 5. 保持连接 ReadLoop（处理断连 & Ping/Pong）
	go readLoop(userId, conn)
}

func readLoop(userId int64, conn *websocket.Conn) {
	defer Hub.Remove(userId)
	for {
		// 只需处理 Close 帧，忽略业务消息
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
