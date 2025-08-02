package config

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaConfig struct {
	Brokers string
	GroupID string
	Topic   string
}

type Producer struct {
	Client *kafka.Producer
	Topic  string
}

type Consumer struct {
	Client *kafka.Consumer
	Topic  string
}

func NewProducer(brokers, topic string) (*Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания продюсера: %w", err)
	}
	return &Producer{Client: p, Topic: topic}, nil
}

func (p *Producer) Close() {
	p.Client.Flush(10000)
	p.Client.Close()
}

func NewConsumer(brokers, groupID, topic string) (*Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"group.id":          groupID,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания консьюмера: %w", err)
	}
	if err := c.Subscribe(topic, nil); err != nil {
		return nil, fmt.Errorf("ошибка подписки консьюмера на топик: %w", err)
	}

	return &Consumer{Client: c, Topic: topic}, nil
}

func (c *Consumer) Close() error {
	return c.Client.Close()
}
