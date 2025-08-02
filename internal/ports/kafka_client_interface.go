package ports

import "context"

type KafkaProducer interface {
	ProduceMessage(topic string, message []byte) error
}

type KafkaConsumer interface {
	ConsumeMessages(ctx context.Context, handler func(msg []byte) error) error
}
