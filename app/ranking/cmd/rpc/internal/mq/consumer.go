package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"freeexchanged/app/ranking/cmd/rpc/internal/constant"
	"freeexchanged/app/ranking/cmd/rpc/internal/svc"
	"freeexchanged/pkg/events"
	"freeexchanged/pkg/eventstream"

	"github.com/zeromicro/go-zero/core/logx"
)

type ArticleConsumer struct {
	ctx                 context.Context
	svcCtx              *svc.ServiceContext
	articleConsumer     *eventstream.Consumer
	interactionConsumer *eventstream.Consumer
}

func NewArticleConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *ArticleConsumer {
	kafkaConf := eventstream.KafkaConf{Brokers: svcCtx.Config.Kafka.Brokers}
	return &ArticleConsumer{
		ctx:                 ctx,
		svcCtx:              svcCtx,
		articleConsumer:     eventstream.NewConsumer(kafkaConf, events.TopicArticleEvents, constant.RankingArticleGroup),
		interactionConsumer: eventstream.NewConsumer(kafkaConf, events.TopicInteractionEvents, constant.RankingInteractionGroup),
	}
}

func (c *ArticleConsumer) Start() {
	logx.Info("Kafka ranking consumer started, waiting for article and interaction events...")

	go c.consumeLoop("article", c.articleConsumer, c.handleArticleMessage)
	go c.consumeLoop("interaction", c.interactionConsumer, c.handleInteractionMessage)
}

func (c *ArticleConsumer) Close() {
	if c.articleConsumer != nil {
		_ = c.articleConsumer.Close()
	}
	if c.interactionConsumer != nil {
		_ = c.interactionConsumer.Close()
	}
}

func (c *ArticleConsumer) consumeLoop(name string, consumer *eventstream.Consumer, handler func([]byte)) {
	for {
		msg, err := consumer.ReadMessage(c.ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || c.ctx.Err() != nil {
				return
			}
			logx.Errorf("Failed to consume %s Kafka event: %v", name, err)
			time.Sleep(time.Second)
			continue
		}
		handler(msg.Value)
	}
}

type PublishEvent struct {
	ArticleID  int64  `json:"article_id"`
	Title      string `json:"title"`
	EventType  string `json:"event_type"`
	OccurredAt int64  `json:"occurred_at"`
}

func (c *ArticleConsumer) handleArticleMessage(body []byte) {
	var event PublishEvent
	if err := json.Unmarshal(body, &event); err != nil {
		logx.Errorf("Error decoding article event: %v", err)
		return
	}

	if event.EventType != events.EventArticlePublished && event.EventType != "publish" {
		return
	}

	score := event.OccurredAt
	if score <= 0 {
		score = time.Now().Unix()
	}

	_, err := c.svcCtx.Redis.ZaddCtx(c.ctx, constant.RankingHotKey, score, fmt.Sprintf("%d", event.ArticleID))
	if err != nil {
		logx.Errorf("Failed to update ranking: %v", err)
		return
	}
	logx.Infof("Added article %d to ranking with score %d", event.ArticleID, score)
}

type InteractionEvent struct {
	UserID     int64  `json:"user_id"`
	ArticleID  int64  `json:"article_id"`
	EventType  string `json:"event_type"`
	Timestamp  int64  `json:"timestamp"`
	OccurredAt int64  `json:"occurred_at"`
}

func (c *ArticleConsumer) handleInteractionMessage(body []byte) {
	var event InteractionEvent
	if err := json.Unmarshal(body, &event); err != nil {
		logx.Errorf("Error decoding interaction event: %v", err)
		return
	}
	if event.ArticleID <= 0 {
		logx.Errorf("Interaction event missing article_id")
		return
	}

	delta := int64(0)
	switch event.EventType {
	case events.EventInteractionLike, "like":
		delta = 10
	case events.EventInteractionUnlike, "unlike":
		delta = -10
	case events.EventInteractionRead, "read":
		delta = 1
	default:
		logx.Errorf("Unknown interaction event type: %s", event.EventType)
		return
	}

	_, err := c.svcCtx.Redis.ZincrbyCtx(c.ctx, constant.RankingHotKey, delta, fmt.Sprintf("%d", event.ArticleID))
	if err != nil {
		logx.Errorf("Failed to update ranking from interaction: %v", err)
		return
	}
	logx.Infof("Applied %s event to article %d with delta %d", event.EventType, event.ArticleID, delta)
}
