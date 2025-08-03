package repository

import (
	"Order_information/internal/config"
	"Order_information/internal/model"
	"Order_information/util"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type CacheRepository struct {
	client *config.RedisClient
	ttl    time.Duration
}

func NewCacheRepository(rdb *config.RedisClient, ttl time.Duration) *CacheRepository {
	return &CacheRepository{rdb, ttl}
}

func (repo *CacheRepository) SetOrder(ctx context.Context, order *model.FullOrder) error {
	data, err := json.Marshal(order)
	if err != nil {
		return util.LogError("ошибка сериализации заказа", err)
	}

	cmd := repo.client.Client.Set(ctx, repo.key(order.Order.OrderUID), data, repo.ttl)
	if err = cmd.Err(); err != nil {
		return util.LogError("ошибка сохранения в Redis", err)
	}
	if cmd.Val() != "OK" {
		return fmt.Errorf("неожиданный ответ Redis: %s", cmd.Val())
	}

	return nil
}

func (repo *CacheRepository) GetOrder(ctx context.Context, uuid string) (*model.FullOrder, error) {
	val, err := repo.client.Client.Get(ctx, repo.key(uuid)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // нет в кэше
	} else if err != nil {
		return nil, util.LogError("ошибка получения заказа из Redis", err)
	}

	var order model.FullOrder
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, util.LogError("ошибка десериализации заказа из кэша", err)
	}
	return &order, nil
}

func (repo *CacheRepository) DeleteOrder(ctx context.Context, uuid string) error {
	if err := repo.client.Client.Del(ctx, repo.key(uuid)).Err(); err != nil {
		return util.LogError("ошибка удаления заказа из Redis", err)
	}
	return nil
}

func (repo *CacheRepository) Pipeline() redis.Pipeliner {
	return repo.client.Client.Pipeline()
}

func (repo *CacheRepository) key(uuid string) string {
	return fmt.Sprintf("order:%s", uuid)
}
