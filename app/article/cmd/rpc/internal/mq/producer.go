package mq

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"freeexchanged/pkg/events"
	"freeexchanged/pkg/eventstream"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type KafkaConf struct {
	Brokers []string
}

type Producer struct {
	writer *eventstream.Producer
}

func NewProducer(c KafkaConf) *Producer {
	return &Producer{
		writer: eventstream.NewProducer(eventstream.KafkaConf{Brokers: c.Brokers}, events.TopicArticleEvents),
	}
}

func (p *Producer) PublishArticleEvent(ctx context.Context, articleID int64, title string, authorID int64) error {
	msg := map[string]any{
		"event_id":    uuid.NewString(),
		"event_type":  events.EventArticlePublished,
		"version":     1,
		"article_id":  articleID,
		"title":       title,
		"author_id":   authorID,
		"occurred_at": time.Now().Unix(),
	}
	body, _ := json.Marshal(msg)

	if err := p.writer.Publish(ctx, strconv.FormatInt(articleID, 10), body); err != nil {
		logx.Errorf("Failed to publish Kafka message: %v", err)
		return err
	}
	logx.Infof("Published article event: %s", string(body))
	return nil
}

func (p *Producer) Close() {
	if p.writer != nil {
		_ = p.writer.Close()
	}
}
