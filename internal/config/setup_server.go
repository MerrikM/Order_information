package config

import (
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
)

type AppConfig struct {
	DatabaseConfig DatabaseConfig `yaml:"databaseConfig"`
	RedisConfig    RedisConfig    `yaml:"redisConfig"`
	ServerAddr     string         `yaml:"serverAddr"`
	KafkaConfig    KafkaConfig    `yaml:"kafka"`
}

func LoadConfig(path string) (*AppConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SetupDatabase(dsn string) (*Database, error) {
	return NewDatabaseConnection("postgres", dsn)
}

func SetupRedis(cfg *RedisConfig) (*RedisClient, error) {
	return NewRedisClient(cfg)
}

func SetupRestServer(addr string) (*http.Server, *chi.Mux) {
	router := chi.NewRouter()

	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	return server, router
}

func SetupKafkaProducer(cfg KafkaConfig) (*Producer, error) {
	return NewProducer(cfg.Producer.Brokers, cfg.Producer.Topic)
}

func SetupKafkaConsumer(cfg KafkaConfig) (*Consumer, error) {
	return NewConsumer(cfg.Consumer.Brokers, cfg.Consumer.GroupID, cfg.Consumer.Topic)
}
