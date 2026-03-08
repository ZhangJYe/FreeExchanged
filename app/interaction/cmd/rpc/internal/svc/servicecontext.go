package svc

import (
	"fmt"

	"freeexchanged/app/interaction/cmd/rpc/internal/config"

	"github.com/streadway/amqp"
)

type ServiceContext struct {
	Config    config.Config
	MqChannel *amqp.Channel
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 连接 RabbitMQ
	url := fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		c.RabbitMQ.Username,
		c.RabbitMQ.Password,
		c.RabbitMQ.Host,
		c.RabbitMQ.Port,
		c.RabbitMQ.VHost,
	)
	conn, err := amqp.Dial(url)
	if err != nil {
		panic(err)
	}

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}

	// 确保 Exchange 存在
	err = ch.ExchangeDeclare(
		"interaction.topic", // name
		"topic",             // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		panic(err)
	}

	return &ServiceContext{
		Config:    c,
		MqChannel: ch,
	}
}
