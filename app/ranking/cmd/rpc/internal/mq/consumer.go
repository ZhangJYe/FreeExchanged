package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"freeexchanged/app/ranking/cmd/rpc/internal/constant"
	"freeexchanged/app/ranking/cmd/rpc/internal/svc"

	"github.com/streadway/amqp"
	"github.com/zeromicro/go-zero/core/logx"
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
		return nil
	}

	ch, err := conn.Channel()
	if err != nil {
		logx.Errorf("Failed to open channel: %v", err)
		_ = conn.Close()
		return nil
	}

	if err := declareTopology(ch); err != nil {
		logx.Errorf("Failed to declare ranking MQ topology: %v", err)
		_ = ch.Close()
		_ = conn.Close()
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

	articleMsgs, err := c.channel.Consume(
		constant.ArticlePublishQueueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logx.Errorf("Failed to register article consumer: %v", err)
		return
	}

	interactionMsgs, err := c.channel.Consume(
		constant.InteractionQueueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logx.Errorf("Failed to register interaction consumer: %v", err)
		return
	}

	logx.Info("ArticleConsumer started, waiting for article and interaction messages...")

	go func() {
		for d := range articleMsgs {
			c.handleArticleMessage(d.Body)
		}
	}()

	go func() {
		for d := range interactionMsgs {
			c.handleInteractionMessage(d.Body, d.RoutingKey)
		}
	}()
}

func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(constant.ArticleExchangeName, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare article exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(constant.ArticlePublishQueueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare article queue: %w", err)
	}
	if err := ch.QueueBind(constant.ArticlePublishQueueName, constant.ArticlePublishRoutingKey, constant.ArticleExchangeName, false, nil); err != nil {
		return fmt.Errorf("bind article queue: %w", err)
	}

	if err := ch.ExchangeDeclare(constant.InteractionExchangeName, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare interaction exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(constant.InteractionQueueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare interaction queue: %w", err)
	}
	if err := ch.QueueBind(constant.InteractionQueueName, constant.InteractionRoutingKey, constant.InteractionExchangeName, false, nil); err != nil {
		return fmt.Errorf("bind interaction queue: %w", err)
	}
	return nil
}

type PublishEvent struct {
	ArticleId int64  `json:"article_id"`
	Title     string `json:"title"`
	EventType string `json:"event_type"`
}

func (c *ArticleConsumer) handleArticleMessage(body []byte) {
	var event PublishEvent
	if err := json.Unmarshal(body, &event); err != nil {
		logx.Errorf("Error decoding message: %v", err)
		return
	}

	if event.EventType != "publish" {
		return
	}

	score := time.Now().Unix()
	_, err := c.svcCtx.Redis.ZaddCtx(c.ctx, constant.RankingHotKey, score, fmt.Sprintf("%d", event.ArticleId))
	if err != nil {
		logx.Errorf("Failed to update ranking: %v", err)
		return
	}
	logx.Infof("Added article %d to ranking with score %d", event.ArticleId, score)
}

type InteractionEvent struct {
	UserId    int64  `json:"user_id"`
	ArticleId int64  `json:"article_id"`
	EventType string `json:"event_type"`
	Timestamp int64  `json:"timestamp"`
}

func (c *ArticleConsumer) handleInteractionMessage(body []byte, routingKey string) {
	var event InteractionEvent
	if err := json.Unmarshal(body, &event); err != nil {
		logx.Errorf("Error decoding interaction message: %v", err)
		return
	}
	if event.ArticleId <= 0 {
		logx.Errorf("Interaction event missing article_id, routing_key=%s", routingKey)
		return
	}

	delta := int64(0)
	switch event.EventType {
	case "like":
		delta = 10
	case "unlike":
		delta = -10
	case "read":
		delta = 1
	default:
		logx.Errorf("Unknown interaction event type: %s", event.EventType)
		return
	}

	_, err := c.svcCtx.Redis.ZincrbyCtx(c.ctx, constant.RankingHotKey, delta, fmt.Sprintf("%d", event.ArticleId))
	if err != nil {
		logx.Errorf("Failed to update ranking from interaction: %v", err)
		return
	}
	logx.Infof("Applied %s event to article %d with delta %d", event.EventType, event.ArticleId, delta)
}
