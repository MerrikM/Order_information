package ports

import (
	"context"
	"github.com/segmentio/kafka-go"
)

type KafkaProducer interface {
	ProduceMessage(ctx context.Context, topic string, key, value []byte) error
}

type KafkaConsumer interface {
	ConsumeMessages(ctx context.Context, handler func(msg kafka.Message) error) error
}
