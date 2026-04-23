package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"freeexchanged/app/ranking/internal/config"
	"freeexchanged/app/ranking/internal/constant"
	"freeexchanged/pkg/events"
	"freeexchanged/pkg/eventstream"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const processedEventTTLSeconds = 7 * 24 * 60 * 60

var applyInteractionScript = redis.NewScript(`
if ARGV[2] ~= "" then
  if redis.call("EXISTS", KEYS[2]) == 1 then
    return 0
  end
end
redis.call("ZINCRBY", KEYS[1], ARGV[1], ARGV[3])
if ARGV[2] ~= "" then
  redis.call("SET", KEYS[2], "1", "EX", ARGV[4])
end
return 1
`)

type Consumer struct {
	ctx                 context.Context
	redis               *redis.Redis
	articleConsumer     *eventstream.Consumer
	interactionConsumer *eventstream.Consumer
}

func NewConsumer(ctx context.Context, c config.Config) *Consumer {
	kafkaConf := eventstream.KafkaConf{Brokers: c.Kafka.Brokers}
	return &Consumer{
		ctx:                 ctx,
		redis:               redis.MustNewRedis(c.BizRedis),
		articleConsumer:     eventstream.NewConsumer(kafkaConf, events.TopicArticleEvents, constant.RankingArticleGroup),
		interactionConsumer: eventstream.NewConsumer(kafkaConf, events.TopicInteractionEvents, constant.RankingInteractionGroup),
	}
}

func (c *Consumer) Start() {
	logx.Info("ranking stream consumer started")

	go c.consumeLoop("article", c.articleConsumer, c.handleArticleMessage)
	go c.consumeLoop("interaction", c.interactionConsumer, c.handleInteractionMessage)
}

func (c *Consumer) Close() {
	if c.articleConsumer != nil {
		_ = c.articleConsumer.Close()
	}
	if c.interactionConsumer != nil {
		_ = c.interactionConsumer.Close()
	}
}

func (c *Consumer) consumeLoop(name string, consumer *eventstream.Consumer, handler func([]byte) error) {
	for {
		msg, err := consumer.FetchMessage(c.ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || c.ctx.Err() != nil {
				return
			}
			logx.Errorf("failed to fetch %s Kafka event: %v", name, err)
			time.Sleep(time.Second)
			continue
		}

		c.processMessage(name, consumer, msg, handler)
	}
}

func (c *Consumer) processMessage(name string, consumer *eventstream.Consumer, msg kafka.Message, handler func([]byte) error) {
	for {
		if err := handler(msg.Value); err != nil {
			if c.ctx.Err() != nil {
				return
			}
			logx.Errorf("failed to process %s Kafka event topic=%s partition=%d offset=%d: %v", name, msg.Topic, msg.Partition, msg.Offset, err)
			time.Sleep(time.Second)
			continue
		}

		if err := consumer.CommitMessages(c.ctx, msg); err != nil {
			if c.ctx.Err() != nil {
				return
			}
			logx.Errorf("failed to commit %s Kafka event topic=%s partition=%d offset=%d: %v", name, msg.Topic, msg.Partition, msg.Offset, err)
			time.Sleep(time.Second)
			continue
		}
		return
	}
}

type PublishEvent struct {
	EventID    string `json:"event_id"`
	ArticleID  int64  `json:"article_id"`
	Title      string `json:"title"`
	EventType  string `json:"event_type"`
	OccurredAt int64  `json:"occurred_at"`
}

func (c *Consumer) handleArticleMessage(body []byte) error {
	var event PublishEvent
	if err := json.Unmarshal(body, &event); err != nil {
		logx.Errorf("error decoding article event: %v", err)
		return nil
	}

	if event.EventType != events.EventArticlePublished && event.EventType != "publish" {
		return nil
	}
	if event.ArticleID <= 0 {
		logx.Errorf("article event missing article_id")
		return nil
	}

	score := event.OccurredAt
	if score <= 0 {
		score = time.Now().Unix()
	}

	if _, err := c.redis.ZaddCtx(c.ctx, constant.RankingHotKey, score, fmt.Sprintf("%d", event.ArticleID)); err != nil {
		return fmt.Errorf("redis zadd: %w", err)
	}
	logx.Infof("added article %d to ranking with score %d", event.ArticleID, score)
	return nil
}

type InteractionEvent struct {
	EventID    string `json:"event_id"`
	UserID     int64  `json:"user_id"`
	ArticleID  int64  `json:"article_id"`
	EventType  string `json:"event_type"`
	Timestamp  int64  `json:"timestamp"`
	OccurredAt int64  `json:"occurred_at"`
}

func (c *Consumer) handleInteractionMessage(body []byte) error {
	var event InteractionEvent
	if err := json.Unmarshal(body, &event); err != nil {
		logx.Errorf("error decoding interaction event: %v", err)
		return nil
	}
	if event.ArticleID <= 0 {
		logx.Errorf("interaction event missing article_id")
		return nil
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
		logx.Errorf("unknown interaction event type: %s", event.EventType)
		return nil
	}

	articleID := fmt.Sprintf("%d", event.ArticleID)
	processedKey := "ranking:event:"
	if event.EventID != "" {
		processedKey += event.EventID
	}

	_, err := c.redis.ScriptRunCtx(
		c.ctx,
		applyInteractionScript,
		[]string{constant.RankingHotKey, processedKey},
		strconv.FormatInt(delta, 10),
		event.EventID,
		articleID,
		strconv.Itoa(processedEventTTLSeconds),
	)
	if err != nil {
		return fmt.Errorf("redis apply interaction script: %w", err)
	}
	logx.Infof("applied %s event to article %d with delta %d", event.EventType, event.ArticleID, delta)
	return nil
}
