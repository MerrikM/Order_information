package main

import (
	"Order_information/internal/config"
	"Order_information/internal/handler"
	"Order_information/internal/repository"
	"Order_information/internal/service"
	"context"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("ошибка загрузки конфига: %v", err)
	}

	database, err := config.SetupDatabase(cfg.DatabaseConfig.DSN)
	if err != nil {
		log.Fatalf("не удалось подключиться к БД: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("ошибка при закрытии БД: %v", err)
		}
	}()

	redisClient, err := config.SetupRedis(&cfg.RedisConfig)
	if err != nil {
		log.Fatalf("ошибка подключения к Redis: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("ошибка при закрытии Redis: %v", err)
		}
	}()

	kafkaConsumer, err := config.SetupKafkaConsumer(cfg.KafkaConfig)
	if err != nil {
		log.Fatalf("не удалось подписаться на топик: %v", err)
	}
	kafkaProducer, err := config.SetupKafkaProducer(cfg.KafkaConfig)
	if err != nil {
		log.Fatalf("не удалось создать продюсера: %v", err)
	}

	orderService := buildOrderService(database, redisClient, kafkaConsumer, kafkaProducer)

	if err := orderService.PreloadCache(ctx); err != nil {
		log.Fatalf("ошибка предзагрузки кеша: %v", err)
	}
	log.Println("кеш загружен, запускаем сервер")

	orderHandler := handler.NewOrderHandler(orderService)

	kafkaDone := make(chan struct{})
	go func() {
		defer close(kafkaDone)
		if err := orderService.KafkaService.ConsumeMessages(ctx, orderService.HandleKafkaMessage); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("Kafka consumer завершился с ошибкой: %v", err)
		} else {
			log.Println("Kafka consumer остановлен")
		}
	}()

	restServer, router := config.SetupRestServer(cfg.ServerAddr)
	router.Get("/order", orderHandler.GetOrderByUUID)

	runServer(ctx, restServer, orderService.KafkaService, kafkaDone)
}

func buildOrderService(
	database *config.Database,
	redis *config.RedisClient,
	kafkaConsumer *config.Consumer,
	kafkaProducer *config.Producer,
) *service.OrderService {
	deliveryRepo := repository.NewDeliveryRepository(database)
	itemsRepo := repository.NewItemsRepository(database)
	paymentRepo := repository.NewPaymentRepository(database)
	orderRepo := repository.NewOrderRepository(database, deliveryRepo, paymentRepo, itemsRepo)
	cacheRepo := repository.NewCacheRepository(redis, 15*time.Minute)
	kafkaService := service.NewKafkaService(kafkaProducer, kafkaConsumer)

	return service.NewOrderService(cacheRepo, orderRepo, kafkaService)
}

func runServer(ctx context.Context, server *http.Server, kafkaService *service.KafkaService, kafkaDone <-chan struct{}) {
	serverErrors := make(chan error, 1)
	go func() {
		log.Println("Сервер запущен на " + server.Addr)
		fmt.Println("____________________________________________________")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("ошибка работы сервера: %v", err)
	case sig := <-signalChannel:
		log.Printf("получен сигнал %v, завершаем работу...", sig)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("ошибка при остановке сервера: %v", err)
	} else {
		log.Println("сервер успешно остановлен")
	}

	if err := kafkaService.Close(); err != nil {
		log.Printf("ошибка при остановке Kafka service: %v", err)
	}
	<-kafkaDone

	log.Println("Работа всех сервисов успешно завершена")
}

//order := &model.FullOrder{
//	Order: &model.Order{
//		OrderUID:          "b563",
//		TrackNumber:       "WBILMTESTTRACK",
//		Entry:             "WBIL",
//		Locale:            "", // если нужно, можно задать значение
//		InternalSignature: "", // если нужно, можно задать значение
//		CustomerID:        "test",
//		DateCreated:       time.Date(2021, 11, 26, 6, 22, 19, 0, time.UTC),
//		DeliveryService:   "meest",
//		ShardKey:          "9",
//		SmID:              99,
//		OofShard:          "1",
//	},
//	Delivery: &model.Delivery{
//		OrderUID: "b563",
//		Name:     "Test Testov",
//		Phone:    "+9720000000",
//		Zip:      "2639809",
//		City:     "Kiryat Mozkin",
//		Address:  "Ploshad Mira 1225",
//		Region:   "Kraiot",
//		Email:    "test@gmail.com",
//	},
//	Payment: &model.Payment{
//		Transaction:  "b563",
//		RequestID:    "",
//		Currency:     "WBPay",
//		Provider:     "Mastercard",
//		Amount:       1817,
//		PaymentDT:    1637907727,
//		Bank:         "Alpha",
//		DeliveryCost: 1500,
//		GoodsTotal:   317,
//		CustomFee:    0,
//		OrderUID:     "b563",
//	},
//	Items: []model.Item{
//		{
//			OrderUID:    "b563",
//			ChrtID:      99349301,
//			TrackNumber: "WBILMTESTTRACK",
//			Price:       453,
//			RID:         "b563",
//			Name:        "Mascaras",
//			Sale:        30,
//			Size:        "0",
//			TotalPrice:  317,
//			NmID:        2389212,
//			Brand:       "Vivienne Sabo",
//			Status:      202,
//		},
//	},
//}
//_ = order

//err = orderRepo.SaveFullOrderTx(context.Background(), order)
//if err != nil {
//	log.Fatalf("ошибка при удалении заказа: %v", err)
//}

//order, err = orderRepo.GetFullOrderTx(context.Background(), "b563")
//if err != nil {
//	log.Fatalf("ошибка при удалении заказа: %v", err)
//}

//order, err = orderService.GetOrderByUUID(context.Background(), "b563")
//if err != nil {
//	log.Fatalf("ошибка при удалении заказа: %v", err)
//}
//
//orderJSON, err := json.MarshalIndent(order, "", "  ")
//if err != nil {
//	log.Fatalf("ошибка сериализации заказа: %v", err)
//}
//fmt.Println(string(orderJSON))
