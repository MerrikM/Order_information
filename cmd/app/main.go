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

	orderService := buildOrderService(database, redisClient, time.Duration(cfg.RedisConfig.TTL)*time.Second, kafkaConsumer, kafkaProducer)

	if err := orderService.PreloadCache(ctx, time.Duration(cfg.RedisConfig.TTL)*time.Second); err != nil {
		log.Fatalf("ошибка предзагрузки кеша: %v", err)
	}
	log.Println("кеш загружен, запускаем сервер")

	orderHandler := handler.NewOrderHandler(orderService)

	kafkaDone := make(chan struct{})
	go func() {
		defer close(kafkaDone)

		handler := orderService.HandleKafkaMessage(cfg.KafkaConfig.Producer.Topic)

		if err := orderService.KafkaService.ConsumeMessages(ctx, handler); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("Kafka consumer завершился с ошибкой: %v", err)
		} else {
			log.Println("Kafka consumer остановлен")
		}
	}()

	restServer, router := config.SetupRestServer(cfg.ServerAddr)
	router.Get("/order/{order_uid}", orderHandler.GetOrderByUUID)

	runServer(ctx, restServer, orderService.KafkaService, kafkaDone)
}

func buildOrderService(
	database *config.Database,
	redis *config.RedisClient,
	ttl time.Duration,
	kafkaConsumer *config.Consumer,
	kafkaProducer *config.Producer,
) *service.OrderService {
	deliveryRepo := repository.NewDeliveryRepository(database)
	itemsRepo := repository.NewItemsRepository(database)
	paymentRepo := repository.NewPaymentRepository(database)
	orderRepo := repository.NewOrderRepository(database, deliveryRepo, paymentRepo, itemsRepo)
	cacheRepo := repository.NewCacheRepository(redis, ttl)
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
