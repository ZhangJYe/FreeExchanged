# Phase 9: 文章服务与事件驱动架构实战

> "文章服务本身不复杂，复杂的是如何与排行榜联动。我们将使用 RabbitMQ 实现这一解耦。"

本阶段我们将实现：
1.  **Article RPC**: 文章的 CRUD 操作，MySQL 持久化。
2.  **Event Driven**: 发布文章后，通过 RabbitMQ 异步通知 Ranking 服务。
3.  **Ranking Consumer**: 订阅文章发布消息，将新文章加入排行榜（初始分值）。

---

# Part 1: 架构与设计

## 1.1 总体架构图

```
用户请求
  ↓ (HTTP)
Gateway
  ↓ (gRPC)
Article RPC  ────(发布成功)────> RabbitMQ (Exchange: article.publish)
  ↓ (SQL)                           │
MySQL (articles 表)                 │ (异步消息)
                                    ↓
                               Ranking RPC (Consumer)
                                    ↓
                               Redis ZSet (初始化排行)
```

**设计亮点 (面试素材)**：
*   **解耦**: Article 服务只管存库，不依赖 Ranking 服务。如果 Ranking 挂了，文章照样能发。
*   **异步**: RabbitMQ 削峰填谷，高并发发文章也不会打垮 Ranking 服务。
*   **最终一致性**: 数据库写入成功后发消息，Ranking 稍后处理，保证数据最终进入排行榜。

---

## 1.2 数据库 Schema

我们只需要一张 `articles` 表。

```sql
CREATE TABLE `articles` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL DEFAULT '' COMMENT '标题',
  `content` text NOT NULL COMMENT '内容',
  `author_id` bigint(20) NOT NULL DEFAULT '0' COMMENT '作者ID',
  `status` tinyint(3) NOT NULL DEFAULT '0' COMMENT '状态 0:草稿 1:发布',
  `like_count` bigint(20) NOT NULL DEFAULT '0' COMMENT '点赞数',
  `view_count` bigint(20) NOT NULL DEFAULT '0' COMMENT '浏览数',
  `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_author` (`author_id`),
  KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 1.3 接口设计

### rpc Article (Internal)

| 方法 | 入参 | 出参 | 描述 |
|:---|:---|:---|:---|
| `Publish` | title, content, author_id | article_id | 发布文章 |
| `GetArticle` | article_id | article_info | 获取详情 |
| `ListArticles` | author_id, page, size | list | 获取列表 |

### RabbitMQ Message (Event)

Topic: `article.events`
Routing Key: `article.publish`

Payload:
```json
{
  "article_id": 101,
  "title": "Go-Zero 实战指南",
  "author_id": 1,
  "create_time": 1708320000
}
```

---

## 1.4 目录结构规划

```
app/article/
├── cmd/rpc/
│   ├── internal/logic/
│   │   ├── publishlogic.go      # 发布 + 发消息
│   │   ├── getarticlelogic.go
│   │   └── listarticleslogic.go
│   ├── internal/mq/             # MQ 生产者封装
│   │   └── producer.go
└── model/                       # goctl 生成的 MySQL Model
```

下一节 → **Part 2: 基础设施搭建 (SQL + Proto)**

---

# Part 2: 基础设施搭建

## 2.1 Step 1: 初始化数据库

执行 SQL 建表语句：

```sql
-- 在 MySQL 客户端执行
CREATE DATABASE IF NOT EXISTS freeexchanged;
USE freeexchanged;

CREATE TABLE IF NOT EXISTS `articles` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL DEFAULT '' COMMENT '标题',
  `content` text NOT NULL COMMENT '内容',
  `author_id` bigint(20) NOT NULL DEFAULT '0' COMMENT '作者ID',
  `status` tinyint(3) NOT NULL DEFAULT '0' COMMENT '状态 0:草稿 1:发布',
  `like_count` bigint(20) NOT NULL DEFAULT '0' COMMENT '点赞数',
  `view_count` bigint(20) NOT NULL DEFAULT '0' COMMENT '浏览数',
  `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_author` (`author_id`),
  KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**生成 Model 代码**：
```powershell
# 在根目录执行
goctl model mysql datasource -url="root:123456@tcp(127.0.0.1:3306)/freeexchanged" -table="articles" -dir="./app/article/model" -style=goZero
```

---

## 2.2 Step 2: 定义 Proto

编辑 `desc/article.proto`：

```protobuf
syntax = "proto3";

package article;
option go_package = "./article";

// --- Model Definition ---
message ArticleInfo {
  int64 id = 1;
  string title = 2;
  string content = 3;
  int64 author_id = 4;
  int64 status = 5;
  int64 like_count = 6;
  int64 view_count = 7;
  int64 create_time = 8;
  int64 update_time = 9;
}

// --- Request/Response ---
message PublishReq {
  string title = 1;
  string content = 2;
  int64 author_id = 3;
}
message PublishResp {
  int64 article_id = 1;
}

message GetArticleReq {
  int64 id = 1;
}
message GetArticleResp {
  ArticleInfo article = 1;
}

message ListArticlesReq {
  int64 author_id = 1;  // 可选，0表示查所有
  int32 page = 2;
  int32 page_size = 3;
}
message ListArticlesResp {
  repeated ArticleInfo articles = 1;
}

service Article {
  rpc Publish(PublishReq) returns(PublishResp);
  rpc GetArticle(GetArticleReq) returns(GetArticleResp);
  rpc ListArticles(ListArticlesReq) returns(ListArticlesResp);
}
```

**生成 RPC 代码**：
```powershell
goctl rpc protoc desc/article.proto --go_out=app/article/cmd/rpc --go-grpc_out=app/article/cmd/rpc --zrpc_out=app/article/cmd/rpc --style=goZero
```

---

## 2.3 Step 3: 配置 RabbitMQ

修改 `app/article/cmd/rpc/etc/article.yaml`，添加 MySQL 和 RabbitMQ 配置：

```yaml
Name: article.rpc
ListenOn: 0.0.0.0:8084  # 新端口

Prometheus:
  Host: 0.0.0.0
  Port: 9096
  Path: /metrics

Telemetry:
  Name: article-rpc
  Endpoint: localhost:4318
  Sampler: 1.0
  Batcher: otlphttp
  OtlpHttpPath: /v1/traces

Consul:
  Host: 127.0.0.1:8500
  Key: article.rpc

DataSource: root:123456@tcp(127.0.0.1:3306)/freeexchanged?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai

RabbitMQ:
  Host: 127.0.0.1
  Port: 5672
  Username: guest
  Password: guest
```

---

**Part 2 小结**：
- 数据库表 `articles` 准备就绪。
- Proto 接口新增了 `ListArticles`。
- YAML 配置了所有依赖（MySQL + RabbitMQ + Consul + Telemetry）。

下一节 → **Part 3: 核心业务逻辑实现 (CRUD + 消息发送)**

---

# Part 3: 核心业务逻辑实现

## 3.1 Step 1: 封装 RabbitMQ Producer

我们不要直接在 Logic 里操作 RabbitMQ 原生 Connection，而是封装一个 `Producer`。

**文件**: `app/article/cmd/rpc/internal/mq/producer.go`

```go
package mq

import (
	"encoding/json"
	"fmt"

	"github.com/streadway/amqp"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	ExchangeName   = "article.events"
	RoutingKeyPub  = "article.publish"
)

type RabbitMqConf struct {
	Host     string
	Port     int
	Username string
	Password string
}

type Producer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewProducer(c RabbitMqConf) *Producer {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/", c.Username, c.Password, c.Host, c.Port)
	conn, err := amqp.Dial(url)
	if err != nil {
		panic(err)
	}

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}

	// 声明 Exchange，确保它存在
	err = ch.ExchangeDeclare(
		ExchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		panic(err)
	}

	return &Producer{
		conn:    conn,
		channel: ch,
	}
}

func (p *Producer) PublishArticleEvent(articleId int64, title string, authorId int64) error {
	msg := map[string]interface{}{
		"article_id": articleId,
		"title":      title,
		"author_id":  authorId,
		"event_type": "publish",
	}
	body, _ := json.Marshal(msg)

	err := p.channel.Publish(
		ExchangeName,  // exchange
		RoutingKeyPub, // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		logx.Errorf("Failed to publish message: %v", err)
		return err
	}
	logx.Infof("Published article event: %s", string(body))
	return nil
}
```

---

## 3.2 Step 2: 更新 Config 和 ServiceContext

**文件**: `app/article/cmd/rpc/internal/config/config.go`

```go
package config

import (
	"github.com/zeromicro/go-zero/zrpc"
	consul "github.com/zeromicro/zero-contrib/zrpc/registry/consul"
)

type Config struct {
	zrpc.RpcServerConf
	Consul     consul.Conf
	DataSource string
	RabbitMQ   struct {
		Host     string
		Port     int
		Username string
		Password string
	}
}
```

**文件**: `app/article/cmd/rpc/internal/svc/servicecontext.go`

```go
package svc

import (
	"freeexchanged/app/article/cmd/rpc/internal/config"
	"freeexchanged/app/article/cmd/rpc/internal/mq"
	"freeexchanged/app/article/model"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config      config.Config
	ArticleModel model.ArticlesModel
	Producer    *mq.Producer
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.DataSource)
	return &ServiceContext{
		Config:       c,
		ArticleModel: model.NewArticlesModel(conn),
		Producer: mq.NewProducer(mq.RabbitMqConf{
			Host:     c.RabbitMQ.Host,
			Port:     c.RabbitMQ.Port,
			Username: c.RabbitMQ.Username,
			Password: c.RabbitMQ.Password,
		}),
	}
}
```

---

## 3.3 Step 3: 实现 PublishLogic (核心)

这是事件驱动的起点：**先存 DB，后发 MQ**。

**文件**: `app/article/cmd/rpc/internal/logic/publishlogic.go`

```go
func (l *PublishLogic) Publish(in *article.PublishReq) (*article.PublishResp, error) {
	// 1. 插入数据库
	newArticle := &model.Articles{
		Title:    in.Title,
		Content:  in.Content,
		AuthorId: in.AuthorId,
		Status:   1, // 默认直接发布
	}

	res, err := l.svcCtx.ArticleModel.Insert(l.ctx, newArticle)
	if err != nil {
		return nil, status.Error(codes.Internal, "DB Insert Failed")
	}

	articleId, _ := res.LastInsertId()

	// 2. 发送 MQ 消息 (异步)
	// 注意：这里没有用事务消息，可能有极低概率 DB 成功但 MQ 失败
	// 面试时可以提：如果追求强一致性，可以用 "本地消息表" 方案
	go func() {
		if err := l.svcCtx.Producer.PublishArticleEvent(articleId, in.Title, in.AuthorId); err != nil {
			logx.Errorf("Failed to send publish event: %v", err)
		}
	}()

	return &article.PublishResp{
		ArticleId: articleId,
	}, nil
}
```

---

## 3.4 Step 4: 实现 GetArticleLogic

```go
func (l *GetArticleLogic) GetArticle(in *article.GetArticleReq) (*article.GetArticleResp, error) {
	a, err := l.svcCtx.ArticleModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "Article not found")
		}
		return nil, err
	}

	return &article.GetArticleResp{
		Article: &article.ArticleInfo{
			Id:         a.Id,
			Title:      a.Title,
			Content:    a.Content,
			AuthorId:   a.AuthorId,
			Status:     int64(a.Status),
			LikeCount:  a.LikeCount,
			ViewCount:  a.ViewCount,
			CreateTime: a.CreateTime.Unix(),
			UpdateTime: a.UpdateTime.Unix(),
		},
	}, nil
}
```

---

## 3.5 Step 5: (可选) 实现 ListArticlesLogic

如果 Model 层没有生成 `FindAll` 方法，需要手动扩展 model（暂略，为了快速跑通流程，先主要关注 Publish）。

---

**Part 3 小结**：
- 封装了 RabbitMQ Producer，支持 Topic Exchange。
- Publish 接口实现了**存库+发消息**的逻辑。
- 采用最简单的异步发送策略，兼顾性能和实现复杂度。

下一节 → **Part 4: 消费端实现 (Ranking 服务更新)**

---

# Part 4: 消费端实现 (Ranking 服务更新)

我们需要修改 Ranking 服务，让它监听 RabbitMQ 消息并更新 Redis 排行榜。

## 4.1 Step 1: 实现 RabbitMQ Consumer

在 `app/ranking/cmd/rpc/internal/mq` 下创建 `consumer.go`。

注意：Ranking 服务之前已经有了 MQ Consumer 的骨架（如果用了 goctl 生成的话），如果没有，我们新建一个。

**文件**: `app/ranking/cmd/rpc/internal/mq/article_consumer.go`

```go
package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"freeexchanged/app/ranking/cmd/rpc/internal/svc"

	"github.com/streadway/amqp"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	ExchangeName  = "article.events"
	QueueName     = "ranking_article_queue"
	RoutingKeyPub = "article.publish"
)

type ArticleConsumer struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	conn   *amqp.Connection
	channel *amqp.Channel
}

func NewArticleConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *ArticleConsumer {
	c := svcCtx.Config.RabbitMQ
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/", c.Username, c.Password, c.Host, c.Port)
	conn, err := amqp.Dial(url)
	if err != nil {
		panic(err)
	}

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}

	// 1. 声明 Exchange (幂等)
	err = ch.ExchangeDeclare(ExchangeName, "topic", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// 2. 声明 Queue
	_, err = ch.QueueDeclare(QueueName, true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// 3. 绑定 Queue 到 Exchange
	err = ch.QueueBind(QueueName, RoutingKeyPub, ExchangeName, false, nil)
	if err != nil {
		panic(err)
	}

	return &ArticleConsumer{
		ctx:     ctx,
		svcCtx:  svcCtx,
		conn:    conn,
		channel: ch,
	}
}

func (c *ArticleConsumer) Start() {
	msgs, err := c.channel.Consume(
		QueueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		logx.Errorf("Failed to register consumer: %v", err)
		return
	}

	logx.Info("ArticleConsumer started, waiting for messages...")

	go func() {
		for d := range msgs {
			c.handleMessage(d.Body)
		}
	}()
}

type PublishEvent struct {
	ArticleId  int64  `json:"article_id"`
	Title      string `json:"title"`
	EventType  string `json:"event_type"`
}

func (c *ArticleConsumer) handleMessage(body []byte) {
	var event PublishEvent
	if err := json.Unmarshal(body, &event); err != nil {
		logx.Errorf("Error decoding message: %v", err)
		return
	}

	if event.EventType == "publish" {
		logx.Infof("Received publish event: article_id=%d", event.ArticleId)
		// 将新文章加入 Redis ZSet，初始分数为当前时间戳
		score := float64(time.Now().Unix())
		_, err := c.svcCtx.Redis.Zadd(c.ctx, "ranking:hot", score, fmt.Sprintf("%d", event.ArticleId))
		if err != nil {
			logx.Errorf("Failed to update ranking: %v", err)
		} else {
			logx.Infof("Added article %d to ranking with score %f", event.ArticleId, score)
		}
	}
}
```

---

## 4.2 Step 2: 在 Ranking 服务启动时加载 Consumer

修改 `app/ranking/cmd/rpc/ranking.go`：

```go
func main() {
    // ... 前面的代码不变 ...

    ctx := svc.NewServiceContext(c)

    // --- 启动消费者 ---
    consumer := mq.NewArticleConsumer(context.Background(), ctx)
    consumer.Start()
    // ------------------

    s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
        // ...
    })
    // ...
}
```

---

## 4.3 Step 3: Gateway 集成 (Part 5)

**更新 `desc/gateway.api`**，添加发布文章接口：

```go
// ... Article Module Types ...
type (
    PublishReq {
        Title   string `json:"title"`
        Content string `json:"content"`
    }
    PublishResp {
        ArticleId int64 `json:"article_id"`
    }
)

@server (
    group:      article
    middleware: PasetoMiddleware // 需要鉴权
    prefix:     v1/article
)
service gateway {
    @doc "发布文章"
    @handler Publish
    post /publish (PublishReq) returns (PublishResp)
}
```

**更新 Gateway Logic**:
```go
func (l *PublishLogic) Publish(req *types.PublishReq) (resp *types.PublishResp, err error) {
    // 从 Context 获取当前用户 ID (Paseto Middleware 注入)
    userId := l.ctx.Value("userId").(json.Number)
    authorId, _ := userId.Int64()

    res, err := l.svcCtx.ArticleRpc.Publish(l.ctx, &article.PublishReq{
        Title:    req.Title,
        Content:  req.Content,
        AuthorId: authorId,
    })
    if err != nil {
        return nil, err
    }

    return &types.PublishResp{
        ArticleId: res.ArticleId,
    }, nil
}
```

---

# Part 5: 验证测试

1.  **启动 Article RPC**: `go run app/article/cmd/rpc/article.go ...`
2.  **重启 Ranking RPC**: 确保它启动了 RabbitMQ Consumer。
3.  **重启 Gateway**: 确保新路由生效。
4.  **测试发布**:
    ```powershell
    # 这里的 TOKEN 需要先登录获取
    curl -X POST http://localhost:8888/v1/article/publish \
      -H "Authorization: Bearer $TOKEN" \
      -d '{"title":"Hello RabbitMQ","content":"Article content..."}'
    ```
5.  **观察日志**:
    *   Article RPC: `Published article event: ...`
    *   Ranking RPC: `Received publish event ... Added article 101 to ranking`
6.  **验证排行榜**:
    ```powershell
    curl http://localhost:8888/v1/ranking/top
    ```
    应该能看到新发布的文章在列表里。

---

**Phase 9 总结**：
我们通过实现 Article 服务和 RabbitMQ 事件驱动，完成了微服务间基于消息的解耦通信。这不仅实现了业务功能，更展示了**事件驱动架构 (EDA)** 的核心思想，这是高级后端工程师必备的技能。



