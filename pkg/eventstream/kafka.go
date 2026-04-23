package eventstream

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

const defaultBroker = "127.0.0.1:9092"

type KafkaConf struct {
	Brokers []string `json:",optional"`
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(c KafkaConf, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(normalizeBrokers(c.Brokers)...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
	if p == nil || p.writer == nil {
		return errors.New("kafka producer is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	})
}

func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(c KafkaConf, topic, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  normalizeBrokers(c.Brokers),
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 1,
			MaxBytes: 10e6,
			MaxWait:  3 * time.Second,
		}),
	}
}

func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if c == nil || c.reader == nil {
		return kafka.Message{}, errors.New("kafka consumer is not initialized")
	}
	return c.reader.FetchMessage(ctx)
}

func (c *Consumer) CommitMessages(ctx context.Context, messages ...kafka.Message) error {
	if c == nil || c.reader == nil {
		return errors.New("kafka consumer is not initialized")
	}
	return c.reader.CommitMessages(ctx, messages...)
}

func (c *Consumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func normalizeBrokers(brokers []string) []string {
	normalized := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		broker = strings.TrimSpace(broker)
		if broker != "" {
			normalized = append(normalized, broker)
		}
	}
	if len(normalized) == 0 {
		return []string{defaultBroker}
	}
	return normalized
}
