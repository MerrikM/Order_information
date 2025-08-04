# Order Service

## Описание
Этот проект представляет собой микросервис на языке Go для обработки заказов. Сервис получает данные заказов из очереди сообщений Kafka, сохраняет их в базу данных PostgreSQL и кэширует в Redis для быстрого доступа. Также предоставляется REST API для получения информации о заказах по их уникальному идентификатору (`order_uid`).

## Основные функции
- **Получение сообщений из Kafka**: Сервис подписывается на топик Kafka, обрабатывает входящие сообщения (события `create`, `update`, `delete`) и выполняет соответствующие действия с заказами.
- **Хранение данных**: Заказы сохраняются в PostgreSQL с разделением данных на таблицы `orders`, `deliveries`, `payments` и `items`.
- **Кэширование**: Данные заказов кэшируются в Redis для быстрого доступа.
- **REST API**: Предоставляется эндпоинт `/order/{order_uid}` для получения информации о заказе по его идентификатору.
- **Обработка ошибок**: Сервис отправляет ответы в Kafka-топик `orders-response` с информацией об успехе или ошибке обработки событий.

## Технологии
- **Язык программирования**: Go
- **База данных**: PostgreSQL
- **Очередь сообщений**: Kafka
- **Кэш**: Redis
- **Веб-фреймворк**: chi (для REST API)
- **Библиотеки**:
    - `github.com/jmoiron/sqlx` для работы с PostgreSQL
    - `github.com/redis/go-redis/v9` для работы с Redis
    - `github.com/segmentio/kafka-go` для работы с Kafka
    - `github.com/go-chi/chi/v5` для маршрутизации HTTP-запросов

## Установка и запуск
### Требования
- Go 1.21 или выше
- PostgreSQL 13+
- Redis 6+
- Kafka 2.8+
- Доступ к конфигурационному файлу `config.yaml`

### Установка
1. **Клонируйте репозиторий**:
   ```bash
   git clone <repository-url>
   cd order-service
   ```

2. **Установите зависимости**:
   ```bash
   go mod tidy
   ```

3. **Настройте конфигурацию**:
   Создайте файл `config.yaml` в корневой директории проекта. Пример конфигурации:
   ```yaml
    databaseConfig:
      dsn: "postgresql://user:password@localhost:5432/postgres?sslmode=disable"
    
    redisConfig:
      address: "localhost:6379"
      password: ""
      database: 0
      ttl: 3600
    
    serverAddr: ":8080"
    
    kafka:
      consumer:
        brokers:
          - "localhost:9092"
        groupID: "order-consumer-group"
        topic: "orders"
    
    producer:
      brokers:
        - "localhost:9092"
      topic: "order-responses"

   ```
    - `database.dsn`: строка подключения к PostgreSQL.
    - `redisConfig`: параметры подключения к Redis (адрес, пароль, база данных, время жизни кэша).
    - `kafka`: параметры Kafka (брокеры, топик для потребления сообщений, топик для отправки ответов).
    - `server_addr`: адрес и порт для REST API.

4. **Настройте базу данных**:
   Схема таблиц БД находится в /init/init_database.sql

5. **Запустите сервис**:
   ```bash
   go run main.go
   ```

### Конфигурация окружения
- Убедитесь, что PostgreSQL, Redis и Kafka запущены и доступны по указанным в `config.yaml` адресам.
- Проверьте, что топик `orders` существует в Kafka для потребления сообщений, и топик `orders-response` доступен для отправки ответов.

## Структура проекта
- **`/cmd/app`**: Точка входа в приложение (`main.go`).
- **`/config`**: Настройка подключений к PostgreSQL, Redis и Kafka.
- **`/model`**: Определения структур данных (`Order`, `FullOrder`, `Delivery`, `Payment`, `Item`).
- **`/service`**: Бизнес-логика сервиса, включая обработку сообщений Kafka и взаимодействие с базой данных и кэшем.
- **`/handler`**: Обработчики HTTP-запросов для REST API.
- **`/repository`**: Репозитории для работы с PostgreSQL и Redis.
- **`/util`**: Утилиты для обработки ошибок и ответов HTTP.

## REST API
### Получение заказа по `order_uid`
- **Эндпоинт**: `GET /order/{order_uid}`
- **Описание**: Возвращает полную информацию о заказе, включая данные о доставке, оплате и товарах.
- **Параметры**:
    - `order_uid`: Уникальный идентификатор заказа (обязательный).
- **Пример запроса**:
  ```bash
  curl http://localhost:8080/order/b563feb7b2b84b6test
  ```
- **Пример ответа**:
  ```json
  {
      "order_uid": "b563feb7b2b84b6test",
      "track_number": "WBILMTESTTRACK",
      "entry": "WBIL",
      "delivery": {
          "name": "Test Testov",
          "phone": "+9720000000",
          "zip": "2639809",
          "city": "Kiryat Mozkin",
          "address": "Ploshad Mira 15",
          "region": "Kraiot",
          "email": "test@gmail.com"
      },
      "payment": {
          "transaction": "b563feb7b2b84b6test",
          "request_id": "",
          "currency": "USD",
          "provider": "wbpay",
          "amount": 1817,
          "payment_dt": 1637907727,
          "bank": "alpha",
          "delivery_cost": 1500,
          "goods_total": 317,
          "custom_fee": 0
      },
      "items": [
          {
              "chrt_id": 9934930,
              "track_number": "WBILMTESTTRACK",
              "price": 453,
              "rid": "ab4219087a764ae0btest",
              "name": "Mascaras",
              "sale": 30,
              "size": "0",
              "total_price": 317,
              "nm_id": 2389212,
              "brand": "Vivienne Sabo",
              "status": 202
          }
      ],
      "locale": "en",
      "internal_signature": "",
      "customer_id": "test",
      "delivery_service": "meest",
      "shardkey": "9",
      "sm_id": 99,
      "date_created": "2021-11-26T06:22:19Z",
      "oof_shard": "1"
  }
  ```
- **Коды ответа**:
    - `200 OK`: Заказ успешно найден.
    - `400 Bad Request`: Отсутствует параметр `order_uid`.
    - `404 Not Found`: Заказ не найден.
    - `500 Internal Server Error`: Ошибка сервера.

## Формат сообщений Kafka
- **Входящие сообщения**:
    - Топик: `orders` (указан в `config.yaml`).
    - Формат: JSON с полем `event` (`create`, `update`, `delete`) и `full_order` (структура `FullOrder`).
    - Пример:
      ```json
      {
          "event": "create",
          "full_order": {
              "order_uid": "b563feb7b2b84b6test",
              "track_number": "WBILMTESTTRACK",
              "entry": "WBIL",
              "delivery": {
                  "name": "Test Testov",
                  "phone": "+9720000000",
                  "zip": "2639809",
                  "city": "Kiryat Mozkin",
                  "address": "Ploshad Mira 15",
                  "region": "Kraiot",
                  "email": "test@gmail.com"
              },
              "payment": {
                  "transaction": "b563feb7b2b84b6test",
                  "request_id": "",
                  "currency": "USD",
                  "provider": "wbpay",
                  "amount": 1817,
                  "payment_dt": 1637907727,
                  "bank": "alpha",
                  "delivery_cost": 1500,
                  "goods_total": 317,
                  "custom_fee": 0
              },
              "items": [
                  {
                      "chrt_id": 9934930,
                      "track_number": "WBILMTESTTRACK",
                      "price": 453,
                      "rid": "ab4219087a764ae0btest",
                      "name": "Mascaras",
                      "sale": 30,
                      "size": "0",
                      "total_price": 317,
                      "nm_id": 2389212,
                      "brand": "Vivienne Sabo",
                      "status": 202
                  }
              ],
              "locale": "en",
              "internal_signature": "",
              "customer_id": "test",
              "delivery_service": "meest",
              "shardkey": "9",
              "sm_id": 99,
              "date_created": "2021-11-26T06:22:19Z",
              "oof_shard": "1"
          }
      }
      ```

- **Исходящие сообщения**:
    - Топик: `orders-response` (указан в `config.yaml`).
    - Формат: JSON с полями `event`, `order_uid`, `timestamp`, и `error` (если произошла ошибка).
    - Пример успешного ответа:
      ```json
      {
          "event": "create_success",
          "order_uid": "b563feb7b2b84b6test",
          "timestamp": "2025-08-04T15:06:00Z"
      }
      ```
    - Пример ответа с ошибкой:
      ```json
      {
          "event": "create_error",
          "order_uid": "b563feb7b2b84b6test",
          "timestamp": "2025-08-04T15:06:00Z",
          "error": "заказ с order_uid b563feb7b2b84b6test уже существует"
      }
      ```

## Обработка ошибок
- **Ошибки Kafka**:
    - Некорректный JSON: Отправляется ответ с `event: "invalid_json"` и текстом ошибки.
    - Неподдерживаемый тип события: Отправляется ответ с `event: "<event>_error"` и описанием ошибки.
- **Ошибки базы данных**:
    - Ошибки сохранения или обновления данных логируются и возвращаются в Kafka-ответе.
- **Ошибки REST API**:
    - Возвращаются HTTP-коды и сообщения об ошибках в формате JSON.

## Логирование
- Все операции (получение сообщений Kafka, сохранение/обновление в базе данных и кэше, ошибки) логируются с помощью `log.Printf`.
- Логи включают информацию о событии, `order_uid`, и данные заказа для отладки.

## Ограничения
- Поле `chrt_id` в таблице `items` должно быть уникальным идентификатором, переданным из JSON. Если требуется автоинкрементное значение, необходимо изменить схему базы данных или добавить дополнительное поле.
- Сервис предполагает, что Kafka-топики и PostgreSQL-таблицы созданы заранее.
- Время жизни кэша в Redis задаётся в `config.yaml` (поле `redis.ttl`).

## Разработка и тестирование
- **Тестирование Kafka**:
    - Отправьте тестовое сообщение в топик `orders` с помощью Kafka-продюсера:
      ```bash
      kafka-console-producer --broker-list localhost:9092 --topic orders < test.json
      ```
    - Проверьте ответы в топике `orders-response`:
      ```bash
      kafka-console-consumer --bootstrap-server localhost:9092 --topic orders-response --from-beginning
      ```

- **Тестирование REST API**:
    - Используйте `curl` или Postman для отправки запросов к эндпоинту `/order/{order_uid}`.
    - Пример:
      ```bash
      curl http://localhost:8080/order/b563feb7b2b84b6test
      ```

- **Логи**:
    - Проверяйте логи в консоли для диагностики ошибок. Пример лога:
      ```
      [Kafka] получен ивент: create для заказа с orderUID: b563feb7b2b84b6test
      [Kafka][create] заказ успешно сохранён, order_uid: b563feb7b2b84b6test
      [Kafka][create] заказ успешно сохранён в кеш, order_uid: b563feb7b2b84b6test
      [Kafka][producer] отправил событие сообщение {event: create_success, order_uid: b563feb7b2b84b6test, timestamp: 2025-08-04T15:06:00Z} в топик orders-response
      ```