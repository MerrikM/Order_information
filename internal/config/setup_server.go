package config

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
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

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8080", "http://127.0.0.1:8080", "http://localhost:63342"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

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
