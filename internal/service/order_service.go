package service

import (
	"Order_information/internal/model"
	"Order_information/internal/repository"
	"Order_information/util"
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

type OrderService struct {
	cacheRepository *repository.CacheRepository
	orderRepository *repository.OrderRepository
	KafkaService    *KafkaService
}

type KafkaResponseEvent struct {
	Event     string `json:"event"`     // "create_error", "create_success" и т.д, отправляются обратно в другой топик для двухсторонней связи
	OrderUID  string `json:"order_uid"` // идентификатор заказа
	Error     string `json:"error"`     // сообщение об ошибке (если есть)
	Timestamp string `json:"timestamp"` // для отладки
}

func NewOrderService(cacheRepository *repository.CacheRepository, orderRepository *repository.OrderRepository, kafkaService *KafkaService) *OrderService {
	return &OrderService{
		cacheRepository: cacheRepository,
		orderRepository: orderRepository,
		KafkaService:    kafkaService,
	}
}

func (s *OrderService) GetOrderByUUID(ctx context.Context, uuid string) (*model.FullOrder, error) {
	order, err := s.cacheRepository.GetOrder(ctx, uuid)
	if err != nil {
		return nil, util.LogError("ошибка при обращении к кэшу", err)
	}

	if order != nil {
		log.Printf("заказ с uuid=%s был взят из кэша Redis", uuid)
		return order, nil
	}

	order, err = s.orderRepository.GetFullOrderByUUIDTx(ctx, uuid)
	if err != nil {
		return nil, util.LogError("ошибка обращения к таблице заказа", err)
	}

	if err := s.cacheRepository.SetOrder(ctx, order); err != nil {
		util.LogError("ошибка Redis", err)
	}

	log.Printf("заказ не был найден в кэше Redis, был выполнен fallback к БД")
	return order, nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, order *model.FullOrder) error {
	err := s.orderRepository.UpdateFullOrderTx(ctx, order)
	if err != nil {
		return util.LogError("не удалось обновить информацию о заказе", err)
	}
	if err = s.cacheRepository.SetOrder(ctx, order); err != nil {
		util.LogError("ошибка Redis", err)
	}

	log.Printf("информация о заказе с uuid=%s обновлена", order.Order.OrderUID)

	return nil
}

func (s *OrderService) DeleteOrderByUUID(ctx context.Context, uuid string) error {
	err := s.orderRepository.DeleteOrderByUUIDTx(ctx, uuid)
	if err != nil {
		return util.LogError("не удалось удалить заказ", err)
	}
	err = s.cacheRepository.DeleteOrder(ctx, uuid)
	if err != nil {
		util.LogError("не удалось заказ из кэша", err)
	}

	return nil
}

