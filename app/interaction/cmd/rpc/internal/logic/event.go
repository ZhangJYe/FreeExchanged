package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"freeexchanged/pkg/events"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type interactionEvent struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	Version    int    `json:"version"`
	UserID     int64  `json:"user_id"`
	ArticleID  int64  `json:"article_id"`
	OccurredAt int64  `json:"occurred_at"`
}

func recordLike(ctx context.Context, conn sqlx.SqlConn, userID, articleID int64) (bool, error) {
	return recordInteractionStateChange(ctx, conn, events.EventInteractionLike, userID, articleID, func(ctx context.Context, session sqlx.Session) (bool, error) {
		res, err := session.ExecCtx(ctx, `
INSERT INTO interaction_states
  (user_id, article_id, liked, create_time, update_time)
VALUES
  (?, ?, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
  liked = IF(liked = 1, liked, VALUES(liked)),
  update_time = IF(liked = 1, update_time, NOW())`, userID, articleID)
		if err != nil {
			return false, err
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return false, err
		}
		return affected > 0, nil
	})
}

func recordUnlike(ctx context.Context, conn sqlx.SqlConn, userID, articleID int64) (bool, error) {
	return recordInteractionStateChange(ctx, conn, events.EventInteractionUnlike, userID, articleID, func(ctx context.Context, session sqlx.Session) (bool, error) {
		res, err := session.ExecCtx(ctx, `
UPDATE interaction_states
SET liked = 0, update_time = NOW()
WHERE user_id = ? AND article_id = ? AND liked = 1`, userID, articleID)
		if err != nil {
			return false, err
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return false, err
		}
		return affected > 0, nil
	})
}

func recordRead(ctx context.Context, conn sqlx.SqlConn, userID, articleID int64) error {
	return conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if _, err := session.ExecCtx(ctx, `
INSERT INTO interaction_states
  (user_id, article_id, liked, read_count, last_read_at, create_time, update_time)
VALUES
  (?, ?, 0, 1, NOW(), NOW(), NOW())
ON DUPLICATE KEY UPDATE
  read_count = read_count + 1,
  last_read_at = NOW(),
  update_time = NOW()`, userID, articleID); err != nil {
			return err
		}

		return enqueueInteractionEvent(ctx, session, events.EventInteractionRead, userID, articleID)
	})
}

func recordInteractionStateChange(ctx context.Context, conn sqlx.SqlConn, eventType string, userID, articleID int64, change func(context.Context, sqlx.Session) (bool, error)) (bool, error) {
	changed := false
	err := conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		ok, err := change(ctx, session)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		changed = true
		return enqueueInteractionEvent(ctx, session, eventType, userID, articleID)
	})
	return changed, err
}

func enqueueInteractionEvent(ctx context.Context, session sqlx.Session, eventType string, userID, articleID int64) error {
	payload, err := json.Marshal(interactionEvent{
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

	_, err = session.ExecCtx(ctx, `
INSERT INTO interaction_outbox_events
  (aggregate_type, aggregate_id, event_type, topic, event_key, payload, status, next_retry_at)
VALUES
  (?, ?, ?, ?, ?, ?, 0, NOW())`,
		"article",
		articleID,
		eventType,
		events.TopicInteractionEvents,
		strconv.FormatInt(articleID, 10),
		string(payload),
	)
	return err
}
