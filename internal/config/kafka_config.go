package config

import (
	"github.com/segmentio/kafka-go"
	"log"
)

type KafkaConfig struct {
	Consumer KafkaConsumerConfig `yaml:"consumer"`
	Producer KafkaProducerConfig `yaml:"producer"`
}

type KafkaConsumerConfig struct {
	Brokers []string `yaml:"brokers"`
	GroupID string   `yaml:"groupID"`
	Topic   string   `yaml:"topic"`
}

type KafkaProducerConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
}

type Producer struct {
	Writer *kafka.Writer
}

type Consumer struct {
	Reader *kafka.Reader
}

func NewProducer(brokers []string, topic string) (*Producer, error) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	log.Printf("Kafka producer создан для топика: %s (%v)", topic, brokers)
	return &Producer{Writer: writer}, nil
}

func (p *Producer) Close() error {
	return p.Writer.Close()
}

func NewConsumer(brokers []string, groupID, topic string) (*Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 1,    // минимальный размер для чтения (1 байт)
		MaxBytes: 10e6, // максимальный размер сообщения (10MB)
	})

	log.Printf("Kafka consumer подключен к топику: %s с groupID: %s (%v)", topic, groupID, brokers)
	return &Consumer{Reader: reader}, nil
}

func (c *Consumer) Close() error {
	return c.Reader.Close()
}
