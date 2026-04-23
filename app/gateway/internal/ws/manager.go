package ws

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 54 * time.Second
	sendBufferSize = 256
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
	userID    int64
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
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
	c := &client{
		userID: userId,
		conn:   conn,
		send:   make(chan []byte, sendBufferSize),
		done:   make(chan struct{}),
	}

	m.mu.Lock()
	old := m.clients[userId]
	m.clients[userId] = c
	total := len(m.clients)
	m.mu.Unlock()

	if old != nil {
		old.close()
	}

	go c.writeLoop(m)
	logx.Infof("[WS] user %d connected, total online: %d", userId, total)
}

func (m *Manager) Remove(userId int64, conn *websocket.Conn) {
	m.mu.Lock()
	current, ok := m.clients[userId]
	if !ok || current.conn != conn {
		m.mu.Unlock()
		return
	}

	delete(m.clients, userId)
	total := len(m.clients)
	m.mu.Unlock()

	current.close()
	logx.Infof("[WS] user %d disconnected, total online: %d", userId, total)
}

func (m *Manager) SendToUser(userId int64, msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		logx.Errorf("[WS] marshal user message failed: %v", err)
		return
	}

	m.mu.RLock()
	c, ok := m.clients[userId]
	m.mu.RUnlock()
	if !ok {
		return
	}

	if !c.enqueue(data) {
		logx.Errorf("[WS] send queue full for user %d", userId)
		m.Remove(userId, c.conn)
	}
}

func (m *Manager) Broadcast(msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		logx.Errorf("[WS] marshal broadcast message failed: %v", err)
		return
	}

	m.mu.RLock()
	snapshot := make(map[int64]*client, len(m.clients))
	for uid, c := range m.clients {
		snapshot[uid] = c
	}
	m.mu.RUnlock()

	for uid, c := range snapshot {
		if !c.enqueue(data) {
			logx.Errorf("[WS] broadcast queue full for user %d", uid)
			m.Remove(uid, c.conn)
		}
	}
}

func (c *client) enqueue(data []byte) bool {
	select {
	case <-c.done:
		return false
	case c.send <- data:
		return true
	default:
		return false
	}
}

func (c *client) writeLoop(m *Manager) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		m.Remove(c.userID, c.conn)
		_ = c.conn.Close()
	}()

	for {
		select {
		case <-c.done:
			_ = c.write(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		case msg := <-c.send:
			if !c.write(websocket.TextMessage, msg) {
				return
			}
		case <-ticker.C:
			if !c.write(websocket.PingMessage, nil) {
				return
			}
		}
	}
}

func (c *client) write(messageType int, payload []byte) bool {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(messageType, payload) == nil
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
	})
}
