package config

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

type RedisClient struct {
	Client *redis.Client
}

func (r *RedisConfig) NewRedisClient() (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     r.Address,
		Password: r.Password,
		DB:       r.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ошибка пинга БД Redis'а: %w", err)
	}

	return &RedisClient{Client: client}, nil
}

func (r *RedisClient) Close() error {
	err := r.Client.Close()
	if err != nil {
		return fmt.Errorf("ошибка закрытия соединения с БД Redis'а: %w", err)
	}

	return nil
}
