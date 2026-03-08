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
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewArticleConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *ArticleConsumer {
	c := svcCtx.Config.RabbitMQ
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/", c.Username, c.Password, c.Host, c.Port)
	conn, err := amqp.Dial(url)
	if err != nil {
		logx.Errorf("Failed to connect to RabbitMQ: %v", err)
		return nil // 允许连接失败，不 panic，方便本地调试
	}

	ch, err := conn.Channel()
	if err != nil {
		logx.Errorf("Failed to open channel: %v", err)
		return nil
	}

	// 1. 声明 Exchange (幂等)
	err = ch.ExchangeDeclare(ExchangeName, "topic", true, false, false, false, nil)
	if err != nil {
		logx.Errorf("Failed to declare exchange: %v", err)
		return nil
	}

	// 2. 声明 Queue
	_, err = ch.QueueDeclare(QueueName, true, false, false, false, nil)
	if err != nil {
		logx.Errorf("Failed to declare queue: %v", err)
		return nil
	}

	// 3. 绑定
	err = ch.QueueBind(QueueName, RoutingKeyPub, ExchangeName, false, nil)
	if err != nil {
		logx.Errorf("Failed to bind queue: %v", err)
		return nil
	}

	return &ArticleConsumer{
		ctx:     ctx,
		svcCtx:  svcCtx,
		conn:    conn,
		channel: ch,
	}
}

func (c *ArticleConsumer) Start() {
	if c.channel == nil {
		return
	}

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
	ArticleId int64  `json:"article_id"`
	Title     string `json:"title"`
	EventType string `json:"event_type"`
}

func (c *ArticleConsumer) handleMessage(body []byte) {
	var event PublishEvent
	if err := json.Unmarshal(body, &event); err != nil {
		logx.Errorf("Error decoding message: %v", err)
		return
	}

	if event.EventType == "publish" {
		logx.Infof("Received publish event: article_id=%d", event.ArticleId)
		// 初始热度设为发布时间（确保新文章能排在前面一点，或者设为 0）
		score := time.Now().Unix()
		// 写入 Ranking 的 ZSet
		_, err := c.svcCtx.Redis.ZaddCtx(c.ctx, "ranking:hot", score, fmt.Sprintf("%d", event.ArticleId))
		if err != nil {
			logx.Errorf("Failed to update ranking: %v", err)
		} else {
			logx.Infof("Added article %d to ranking with score %f", event.ArticleId, score)
		}
	}
}
