package svc

import (
	"freeexchanged/app/interaction/cmd/rpc/internal/config"
	"freeexchanged/pkg/events"
	"freeexchanged/pkg/eventstream"
)

type ServiceContext struct {
	Config        config.Config
	EventProducer *eventstream.Producer
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:        c,
		EventProducer: eventstream.NewProducer(eventstream.KafkaConf{Brokers: c.Kafka.Brokers}, events.TopicInteractionEvents),
	}
}
