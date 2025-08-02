package repository

import (
	"Order_information/internal/config"
	"Order_information/internal/model"
	"Order_information/util"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
)

type OrderRepository struct {
	*config.Database
	deliveryRepo *DeliveryRepository
	paymentRepo  *PaymentRepository
	itemsRepo    *ItemsRepository
}

func NewOrderRepository(database *config.Database,
	deliveryRepo *DeliveryRepository,
	paymentRepo *PaymentRepository,
	itemsRepo *ItemsRepository) *OrderRepository {
	return &OrderRepository{
		Database:     database,
		deliveryRepo: deliveryRepo,
		paymentRepo:  paymentRepo,
		itemsRepo:    itemsRepo,
	}
}

func (repo *OrderRepository) GetOrderById(ctx context.Context, id string) (*model.Order, error) {
	query := `SELECT * FROM orders WHERE order_uid=$1`

	var returnedOrder model.Order
	err := repo.GetContext(ctx, &returnedOrder, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.LogError("не удалось вставить данные в таблицу", err)
		}
		return nil, util.LogError("ошибка получения таблицы", err)
	}

	return &returnedOrder, nil
}

func (repo *OrderRepository) SaveFullOrderTx(ctx context.Context, order *model.FullOrder) error {
	transaction, err := repo.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			transaction.Rollback()
			panic(r)
		}
	}()

	if err := repo.SaveOrder(ctx, transaction, order.Order); err != nil {
		transaction.Rollback()
		return fmt.Errorf("не удалось выполнить транзакцию для заказа: %w", err)
	}
	if err := repo.deliveryRepo.SaveDelivery(ctx, transaction, order.Delivery); err != nil {
		transaction.Rollback()
		return fmt.Errorf("не удалось выполнить транзакцию для доставки: %w", err)
	}
	if err := repo.paymentRepo.SavePayment(ctx, transaction, order.Payment); err != nil {
		transaction.Rollback()
		return fmt.Errorf("не удалось выполнить транзакцию для оплаты: %w", err)
	}
	if err := repo.itemsRepo.SaveItems(ctx, transaction, order.Items); err != nil {
		transaction.Rollback()
		return fmt.Errorf("не удалось выполнить транзакцию для товара(ов): %w", err)
	}

	return transaction.Commit()
}

func (repo *OrderRepository) SaveOrder(ctx context.Context, exec sqlx.ExtContext, order *model.Order) error {
	query := `INSERT INTO orders 
    (order_uid, track_number, entry, date_created, delivery_service, shardkey, sm_id, oof_shard) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := exec.ExecContext(
		ctx,
		query,
		order.OrderUID,
		order.TrackNumber,
		order.Entry,
		order.DateCreated,
		order.DeliveryService,
		order.ShardKey,
		order.SmID,
		order.OofShard,
	)

	if err != nil {
		return util.LogError("ошибка при вставке заказа", err)
	}

	return nil
}

