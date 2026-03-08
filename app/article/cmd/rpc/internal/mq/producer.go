package mq

import (
	"encoding/json"
	"fmt"

	"github.com/streadway/amqp"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	ExchangeName  = "article.events"
	RoutingKeyPub = "article.publish"
)

type RabbitMqConf struct {
	Host     string
	Port     int
	Username string
	Password string
}

type Producer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewProducer(c RabbitMqConf) *Producer {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/", c.Username, c.Password, c.Host, c.Port)
	conn, err := amqp.Dial(url)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to RabbitMQ: %v", err))
	}

	ch, err := conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("Failed to open a channel: %v", err))
	}

	// 声明 Exchange，确保它存在 (幂等操作)
	err = ch.ExchangeDeclare(
		ExchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to declare an exchange: %v", err))
	}

	return &Producer{
		conn:    conn,
		channel: ch,
	}
}

func (p *Producer) PublishArticleEvent(articleId int64, title string, authorId int64) error {
	msg := map[string]interface{}{
		"article_id": articleId,
		"title":      title,
		"author_id":  authorId,
		"event_type": "publish",
	}
	body, _ := json.Marshal(msg)

	err := p.channel.Publish(
		ExchangeName,  // exchange
		RoutingKeyPub, // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		logx.Errorf("Failed to publish message: %v", err)
		return err
	}
	logx.Infof("Published article event: %s", string(body))
	return nil
}

func (p *Producer) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
