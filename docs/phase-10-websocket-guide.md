# Phase 10: WebSocket 实时消息推送实战

> "HTTP 是请求-响应，WebSocket 是全双工通信。我们将构建一个基于 Redis Pub/Sub 的实时推送系统。"

本阶段我们将实现：
1.  **Gateway WS Server**: 在 Gateway 中集成 WebSocket 服务。
2.  **PASETO Auth**: 握手阶段鉴权，复用现有 Token 机制。
3.  **Redis Pub/Sub**: 用于服务间消息分发（Interaction/Ranking -> Redis -> Gateway -> User）。

---

# Part 1: 架构与设计

## 1.1 总体架构图

```
User A (Browser)      User B (Browser)
      ▲                     ▲
      │ (WS Connect)        │ (WS Connect)
      │                     │
┌─────┴─────────────────────┴─────┐
│           Gateway (WS)          │
│  [Map: userId -> *ws.Conn]      │ <-- 维护长连接
└──────────────▲──────────────────┘
               │ (Subscribe)
               │
      ┌────────┴────────┐
      │  Redis Pub/Sub  │  <-- 消息总线
      └────────▲────────┘
               │ (Publish)
               │
  ┌────────────┴────────────┐
  │ Interaction RPC / Job   │
  └─────────────────────────┘
  (业务触发: 点赞/榜单更新)
```

**设计亮点 (面试素材)**：
*   **无状态扩张**: Gateway 虽持有连接状态，但通过 Redis Pub/Sub 解耦消息源。任意业务服务只需往 Redis 发消息，不需要知道通过哪个 Gateway 实例推送。
*   **连接管理**: 维护 `ConcurrentMap` 存储在线用户连接，支持心跳检测（Ping/Pong）。
*   **鉴权复用**: WebSocket 握手请求携带 Token，复用 PASETO 逻辑，安全可靠。

---

## 1.2 消息协议设计

### 1. 客户端 -> 服务端 (Upstream)
我们主要做**单向推送**（Server -> Client），客户端只发 **Ping** 心跳。

### 2. 服务端 -> 客户端 (Downstream)
统一 JSON 格式：

```json
{
  "type": "interaction", // or "ranking", "system"
  "payload": {
    "action": "like",
    "from_user_id": 101,
    "title": "Your article got a like!",
    "timestamp": 1708320000
  }
}
```

---

## 1.3 Redis Channel 设计

| Channel | 用途 | Payload 示例 |
|:---|:---|:---|
| `ws:user:notify` | 针对特定用户的通知 (点赞/评论) | `{"target_id": 1, "msg": ...}` |
| `ws:broadcast` | 全员广播 (如排行榜更新) | `{"type": "ranking", "data": ...}` |

---

## 1.4 目录结构规划

```
app/gateway/
├── internal/websocket/
│   ├── manager.go       # 连接管理器 (Add, Remove, Broadcast, SendToUser)
│   ├── handler.go       # WS 握手处理 (Upgrade, Auth)
│   └── listener.go      # Redis Pub/Sub 监听器
```

下一节 → **Part 2: 基础设施与 WebSocket Manager 实现**

---

# Part 2: 基础设施与 WebSocket Manager 实现

我们需要在 Gateway 中实现 WebSocket 的连接管理。

## 2.1 依赖引入

在项目根目录运行：
```powershell
go get github.com/gorilla/websocket
```

---

## 2.2 WebSocket Manager (并发安全)

**核心挑战**：Gateway 是多协程处理请求，而 `map` 是非线程安全的。我们需要用 `sync.RWMutex` 或 `sync.Map`。

**文件**: `app/gateway/internal/websocket/manager.go`

```go
package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	// WebSocket 升级配置 (允许跨域)
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	// 存储在线用户连接: userId -> *websocket.Conn
	clients map[int64]*websocket.Conn
	lock    sync.RWMutex
}

// 单例模式，全局唯一
var Hub = &Manager{
	clients: make(map[int64]*websocket.Conn),
}

// --- Connection Management ---

func (m *Manager) AddClient(userId int64, conn *websocket.Conn) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 如果该用户已有旧连接，先通过 Close 关闭（这里采取"踢出旧设备"策略）
	if oldConn, ok := m.clients[userId]; ok {
		oldConn.Close()
	}
	m.clients[userId] = conn
	logx.Infof("[WS] User %d connected. Total online: %d", userId, len(m.clients))
}

func (m *Manager) RemoveClient(userId int64) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if conn, ok := m.clients[userId]; ok {
		conn.Close()
		delete(m.clients, userId)
		logx.Infof("[WS] User %d disconnected. Total online: %d", userId, len(m.clients))
	}
}

// --- Message Sending ---

// SendToUser: 推送消息给指定用户
func (m *Manager) SendToUser(userId int64, message interface{}) {
	m.lock.RLock()
	conn, ok := m.clients[userId]
	m.lock.RUnlock()

	if !ok {
		// 用户不在线，直接忽略 (生产环境可存入离线消息表)
		return
	}

	payload, _ := json.Marshal(message)
	if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		logx.Errorf("[WS] Failed to send to user %d: %v", userId, err)
		m.RemoveClient(userId) // 发送失败认为掉线
	}
}

// Broadcast: 广播给所有人 (如排行榜更新)
func (m *Manager) Broadcast(message interface{}) {
	payload, _ := json.Marshal(message)

	m.lock.RLock()
	defer m.lock.RUnlock()

	for userId, conn := range m.clients {
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			logx.Errorf("[WS] Failed to broadcast to user %d: %v", userId, err)
			// 注意：遍历时不宜直接 Remove，可能会有锁的问题，这里简单记日志
			conn.Close() 
			// 真正移除可以在下次 Add 或专门的清理协程做
		}
	}
}
```

---

## 2.3 Handshake Handler (Upgrade + Auth)

**文件**: `app/gateway/internal/websocket/handler.go`

```go
package websocket

import (
	"encoding/json"
	"net/http"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/pkg/token"

	"github.com/zeromicro/go-zero/core/logx"
)

// HandleWS: 处理 WebSocket 握手请求
func HandleWS(svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) {
	// 1. 获取 Token (Query Param: ?token=xxx)
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	// 2. 验证 Token (复用 svcCtx.TokenMaker)
	// Paseto Verify 不需要查库，纯计算，很快
	payload, err := svcCtx.TokenMaker.VerifyToken(tokenStr)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userId := payload.UserId

	// 3. 升级协议 HTTP -> WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Errorf("[WS] Upgrade failed: %v", err)
		return
	}

	// 4. 注册连接
	Hub.AddClient(userId, conn)

	// 5. 保持连接并处理 Ping/Pong (ReadLoop)
	// 这一步会阻塞，直到连接断开
	go readLoop(userId, conn)
}

func readLoop(userId int64, conn *websocket.Conn) {
	defer Hub.RemoveClient(userId)

	for {
		// 我们不需要读客户端发来的业务消息，只需要处理 Close 和 Ping
		// 如果读出错，说明连接断开
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
```

---

## 2.4 在 Gateway 中注册路由

我们需要在 Gateway 启动时注册 `/ws` 路由。

**修改**: `app/gateway/gateway.go`

```go
import (
    "freeexchanged/app/gateway/internal/websocket" // 引入包
)

func main() {
    // ...
    server := rest.MustNewServer(c.RestConf)
    defer server.Stop()

    ctx := svc.NewServiceContext(c)
    handler.RegisterHandlers(server, ctx)

    // *** 新增 WebSocket 路由 ***
    // 注意：go-zero 的 engine 支持 AddRoute，但更简单的是在这个 server 上加
    // 但 rest.Server 封装了 http.Server。go-zero 允许通过 Engine().AddRoute 加路由
    // 这里我们用一种 hack 方式：直接定义在 api 文件里是不行的，因为 handler 签名不一样
    // 
    // 正确做法：在 api 里定义一个 get /ws，然后在 handler 里调 websocket.HandleWS
    //
    // 我们将在 Part 3 详细说明如何通过 api 定义这个路由。
}
```

**Wait**：其实最简单也是最规范的做法是：**在 `gateway.api` 里加一个 `/ws` 的 GET 路由**，然后生成 Handler，在 Handler 里调用 `websocket.HandleWS`。这样完全符合 go-zero 规范！

下一节 → **Part 3: API 定义与 Redis 监听实现**

