package ws

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     sameOrigin,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Manager struct {
	clients map[int64]*client
	mu      sync.RWMutex
}

type client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

var Hub = &Manager{
	clients: make(map[int64]*client),
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func (m *Manager) Add(userId int64, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if old, ok := m.clients[userId]; ok {
		_ = old.conn.Close()
	}
	m.clients[userId] = &client{conn: conn}
	logx.Infof("[WS] user %d connected, total online: %d", userId, len(m.clients))
}

func (m *Manager) Remove(userId int64, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.clients[userId]
	if !ok || current.conn != conn {
		return
	}

	_ = current.conn.Close()
	delete(m.clients, userId)
	logx.Infof("[WS] user %d disconnected, total online: %d", userId, len(m.clients))
}

func (m *Manager) SendToUser(userId int64, msg any) {
	m.mu.RLock()
	c, ok := m.clients[userId]
	m.mu.RUnlock()
	if !ok {
		return
	}

	data, _ := json.Marshal(msg)
	c.mu.Lock()
	err := c.conn.WriteMessage(websocket.TextMessage, data)
	c.mu.Unlock()
	if err != nil {
		logx.Errorf("[WS] send to user %d failed: %v", userId, err)
		m.Remove(userId, c.conn)
	}
}

func (m *Manager) Broadcast(msg any) {
	data, _ := json.Marshal(msg)

	m.mu.RLock()
	snapshot := make(map[int64]*client, len(m.clients))
	for uid, c := range m.clients {
		snapshot[uid] = c
	}
	m.mu.RUnlock()

	for uid, c := range snapshot {
		c.mu.Lock()
		err := c.conn.WriteMessage(websocket.TextMessage, data)
		c.mu.Unlock()
		if err != nil {
			logx.Errorf("[WS] broadcast to user %d failed: %v", uid, err)
			m.Remove(uid, c.conn)
		}
	}
}
