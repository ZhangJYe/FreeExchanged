package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"freeexchanged/pkg/eventstream"

	"github.com/google/uuid"
)

type interactionEvent struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	Version    int    `json:"version"`
	UserID     int64  `json:"user_id"`
	ArticleID  int64  `json:"article_id"`
	OccurredAt int64  `json:"occurred_at"`
}

func publishInteractionEvent(ctx context.Context, producer *eventstream.Producer, eventType string, userID, articleID int64) error {
	body, err := json.Marshal(interactionEvent{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		Version:    1,
		UserID:     userID,
		ArticleID:  articleID,
		OccurredAt: time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	return producer.Publish(ctx, strconv.FormatInt(articleID, 10), body)
}
