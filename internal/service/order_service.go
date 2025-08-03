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

func (s *OrderService) UpdateOrder(ctx context.Context, order *model.Order, fullOrder *model.FullOrder) error {
	err := s.orderRepository.UpdateFullOrderTx(ctx, order, fullOrder)
	if err != nil {
		return util.LogError("не удалось обновить информацию о заказе", err)
	}
	if err = s.cacheRepository.SetOrder(ctx, fullOrder); err != nil {
		util.LogError("ошибка Redis", err)
	}

	log.Printf("информация о заказе с uuid=%s обновлена", order.OrderUID)

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

func (s *OrderService) PreloadCache(ctx context.Context) error {
	orders, err := s.orderRepository.GetAllOrders(ctx)
	if err != nil {
		return fmt.Errorf("ошибка при получении заказов из БД: %w", err)
	}

	pipe := s.cacheRepository.Pipeline()

	for _, order := range orders {
		value, err := json.Marshal(order)
		if err != nil {
			log.Printf("не удалось сериализовать заказ с uuid: %s: %v", order.OrderUID, err)
			continue
		}
		pipe.Set(ctx, "order:"+order.OrderUID, value, 0)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("ошибка при загрузке заказов в кэш через pipeline: %w", err)
	}

	log.Printf("Успешно загружен(о) %d заказ(ов) в кэш\n", len(orders))
	return nil
}

func (s *OrderService) HandleKafkaMessage(msg kafka.Message) error {
	var event KafkaEvent

	if err := json.Unmarshal(msg.Value, &event); err != nil {
		resp := KafkaResponseEvent{
			Event:     "invalid_json",
			OrderUID:  "",
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		_ = s.sendKafkaResponse("orders-response", resp)
		return util.LogError("[Kafka] invalid JSON in message", err)
	}

	log.Printf("[Kafka] получен ивент: %s для заказа с orderUID: %s", event.Event, event.FullOrder.OrderUID)

	if event.FullOrder == nil {
		return util.LogError("[Kafka] ERROR", fmt.Errorf("поле FullOrder пустое (nil)"))
	}
	if event.FullOrder.OrderUID == "" {
		return util.LogError("[Kafka] ERROR", fmt.Errorf("поле OrderUID пустое"))
	}
	if event.FullOrder.Delivery == nil {
		return util.LogError("[Kafka] ERROR", fmt.Errorf("поле Delivery пустое (nil)"))
	}
	if event.FullOrder.Payment == nil {
		return util.LogError("[Kafka] ERROR", fmt.Errorf("поле Payment пустое (nil)"))
	}
	if len(event.FullOrder.Items) == 0 {
		return util.LogError("[Kafka] ERROR", fmt.Errorf("список Items пустой"))
	}
	if event.FullOrder.DateCreated.IsZero() {
		return util.LogError("[Kafka] ERROR", fmt.Errorf("поле DateCreated пустое или невалидное"))
	}

	event.FullOrder.Delivery.OrderUID = event.FullOrder.OrderUID
	event.FullOrder.Payment.OrderUID = event.FullOrder.OrderUID
	for i := range event.FullOrder.Items {
		event.FullOrder.Items[i].OrderUID = event.FullOrder.OrderUID
	}

	order := &model.Order{
		OrderUID:          event.FullOrder.OrderUID,
		TrackNumber:       event.FullOrder.TrackNumber,
		Entry:             event.FullOrder.Entry,
		Locale:            event.FullOrder.Locale,
		InternalSignature: event.FullOrder.InternalSignature,
		CustomerID:        event.FullOrder.CustomerID,
		DeliveryService:   event.FullOrder.DeliveryService,
		ShardKey:          event.FullOrder.ShardKey,
		SmID:              event.FullOrder.SmID,
		DateCreated:       event.FullOrder.DateCreated,
		OofShard:          event.FullOrder.OofShard,
	}

	orderUID := event.FullOrder.OrderUID
	var err error

	switch event.Event {
	case "create":
		var exists bool
		exists, err = s.orderRepository.Exists(orderUID)
		if err != nil {
			err = util.LogError("[Kafka][create] ошибка проверки существования заказа", err)
			break
		}
		if exists {
			err = util.LogError("[Kafka][create] заказ уже существует", fmt.Errorf("order_uid %s", orderUID))
			break
		}
		log.Printf("[Kafka][create] вызываю SaveFullOrderTx")
		err = s.orderRepository.SaveFullOrderTx(context.Background(), order, event.FullOrder)
		if err != nil {
			err = util.LogError("[Kafka][create] ошибка сохранения заказа", err)
			break
		}
		log.Printf("[Kafka][create] заказ успешно сохранён, order_uid: %s", orderUID)

		if err = s.cacheRepository.SetOrder(context.Background(), event.FullOrder); err != nil {
			util.LogError("[Kafka][create] ошибка кеширования заказа", err)
		} else {
			log.Printf("[Kafka][create] заказ успешно сохранён в кеш, order_uid: %s", orderUID)
		}

	case "update":
		err = s.orderRepository.UpdateFullOrderTx(context.Background(), order, event.FullOrder)
		if err != nil {
			err = util.LogError("[Kafka][update] ошибка обновления заказа", err)
			break
		}
		log.Printf("[Kafka][update] заказ успешно обновлён, order_uid: %s", orderUID)

		if err = s.cacheRepository.SetOrder(context.Background(), event.FullOrder); err != nil {
			util.LogError("[Kafka][update] ошибка кеширования заказа", err)
		} else {
			log.Printf("[Kafka][update] заказ успешно обновлён в кеше, order_uid: %s", orderUID)
		}

	case "delete":
		err = s.orderRepository.DeleteOrderByUUIDTx(context.Background(), orderUID)
		if err != nil {
			err = util.LogError("[Kafka][delete] ошибка удаления заказа", err)
			break
		}
		log.Printf("[Kafka][delete] заказ успешно удалён, order_uid: %s", orderUID)

		if err = s.cacheRepository.DeleteOrder(context.Background(), orderUID); err != nil {
			util.LogError("[Kafka][delete] ошибка удаления заказа из кеша", err)
		} else {
			log.Printf("[Kafka][delete] заказ успешно удалён из кеша, order_uid: %s", orderUID)
		}

	default:
		err = util.LogError("[Kafka] неподдерживаемый тип события", fmt.Errorf("%s", event.Event))
	}

	response := KafkaResponseEvent{
		Event:     fmt.Sprintf("%s_success", event.Event),
		OrderUID:  orderUID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if err != nil {
		response.Event = fmt.Sprintf("%s_error", event.Event)
		response.Error = err.Error()
	}

	_ = s.sendKafkaResponse("orders-response", response)
	return err
}

func (s *OrderService) sendKafkaResponse(topic string, resp KafkaResponseEvent) error {
	value, err := json.Marshal(resp)
	if err != nil {
		log.Printf("не удалось сериализовать Kafka-ответ: %v", err)
		return err
	}
	return s.KafkaService.ProduceMessage(context.Background(), topic, nil, value)
}
