package service

import (
	"Order_information/internal/config"
	"Order_information/internal/model"
	"context"
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

type KafkaService struct {
	producer *config.Producer
	consumer *config.Consumer
}

type KafkaEvent struct {
	Event     string           `json:"event"`
	FullOrder *model.FullOrder `json:"order"`
}

func NewKafkaService(producer *config.Producer, consumer *config.Consumer) *KafkaService {
	return &KafkaService{
		producer: producer,
		consumer: consumer,
	}
}

func (service *KafkaService) ProduceMessage(ctx context.Context, topic string, key, value []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	}

	err := service.producer.Writer.WriteMessages(ctx, msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения: %w", err)
	}
	return nil
}

func (service *KafkaService) ConsumeMessages(ctx context.Context, handler func(msg kafka.Message) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := service.consumer.Reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			log.Printf("ошибка чтения сообщения из Kafka: %v\n", err)
			continue
		}

		retryErr := retry(ctx, 3, time.Second, func() error {
			return handler(msg)
		})

		if retryErr != nil {
			log.Printf("ошибка после всех попыток обработки сообщения: %v\n", retryErr)
			continue
		}
	}
}

func retry(ctx context.Context, attempts int, delay time.Duration, fn func() error) error {
	for i := 0; i < attempts; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := fn(); err != nil {
			if i == attempts-1 {
				return err
			}
			select {
			case <-time.After(delay * time.Duration(i+1)):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}
		return nil
	}
	return nil
}
