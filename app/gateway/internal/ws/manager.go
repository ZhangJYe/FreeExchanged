package ws

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Manager 管理所有在线 WebSocket 连接
type Manager struct {
	clients map[int64]*websocket.Conn
	mu      sync.RWMutex
}

// Hub 全局单例
var Hub = &Manager{
	clients: make(map[int64]*websocket.Conn),
}

func (m *Manager) Add(userId int64, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 踢出旧连接
	if old, ok := m.clients[userId]; ok {
		old.Close()
	}
	m.clients[userId] = conn
	logx.Infof("[WS] user %d connected, total online: %d", userId, len(m.clients))
}

func (m *Manager) Remove(userId int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn, ok := m.clients[userId]; ok {
		conn.Close()
		delete(m.clients, userId)
		logx.Infof("[WS] user %d disconnected, total online: %d", userId, len(m.clients))
	}
}

// SendToUser 推送给指定用户
func (m *Manager) SendToUser(userId int64, msg any) {
	m.mu.RLock()
	conn, ok := m.clients[userId]
	m.mu.RUnlock()
	if !ok {
		return
	}
	data, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logx.Errorf("[WS] send to user %d failed: %v", userId, err)
		m.Remove(userId)
	}
}

// Broadcast 广播给所有在线用户
func (m *Manager) Broadcast(msg any) {
	data, _ := json.Marshal(msg)
	m.mu.RLock()
	defer m.mu.RUnlock()
	for uid, conn := range m.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			logx.Errorf("[WS] broadcast to user %d failed: %v", uid, err)
		}
	}
}
