package logic

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

const interactionExchange = "interaction.topic"

type interactionEvent struct {
	UserId    int64  `json:"user_id"`
	ArticleId int64  `json:"article_id"`
	EventType string `json:"event_type"`
	Timestamp int64  `json:"timestamp"`
}

func publishInteractionEvent(ch *amqp.Channel, eventType string, userID, articleID int64) error {
	body, err := json.Marshal(interactionEvent{
		UserId:    userID,
		ArticleId: articleID,
		EventType: eventType,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	return ch.Publish(
		interactionExchange,
		fmt.Sprintf("article.%s", eventType),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}
