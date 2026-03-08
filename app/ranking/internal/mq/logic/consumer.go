package logic

import (
	"context"
)

type Consumer struct {
}

func NewConsumer() *Consumer {
	return &Consumer{}
}

func (c *Consumer) Consume(ctx context.Context, key, value string) error {
	// Process message
	return nil
}
