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
	"log"
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

func (repo *OrderRepository) GetFullOrderByUUIDTx(ctx context.Context, orderUID string) (*model.FullOrder, error) {
	transaction, err := repo.BeginTxx(ctx, nil)
	if err != nil {
		return nil, util.LogError("не удалось начать транзакцию", err)
	}
	defer transaction.Rollback()

	var fullOrder model.FullOrder

	order, err := repo.GetOrderByUUID(ctx, transaction, orderUID)
	if err != nil {
		return nil, err
	}
	fullOrder.Order = order

	deliveryRepo := DeliveryRepository{Database: repo.Database}
	delivery, err := deliveryRepo.GetDeliveryByOrderUID(ctx, transaction, orderUID)
	if err != nil {
		return nil, err
	}
	fullOrder.Delivery = delivery

	paymentRepo := PaymentRepository{Database: repo.Database}
	payment, err := paymentRepo.GetPaymentByOrderUID(ctx, transaction, orderUID)
	if err != nil {
		return nil, err
	}
	fullOrder.Payment = payment

	itemsRepo := ItemsRepository{Database: repo.Database}
	items, err := itemsRepo.GetItemsByOrderUID(ctx, transaction, orderUID)
	if err != nil {
		return nil, err
	}
	fullOrder.Items = items

	if err := transaction.Commit(); err != nil {
		return nil, util.LogError("не удалось зафиксировать транзакцию", err)
	}

	return &fullOrder, nil
}

func (repo *OrderRepository) GetAllOrders(ctx context.Context) ([]model.FullOrder, error) {
	var orders []*model.Order
	orderQuery := `SELECT * FROM orders`
	if err := repo.SelectContext(ctx, &orders, orderQuery); err != nil {
		return nil, fmt.Errorf("ошибка при получении заказов: %w", err)
	}

	var deliveries []*model.Delivery
	deliveryQuery := `SELECT * FROM deliveries`
	if err := repo.SelectContext(ctx, &deliveries, deliveryQuery); err != nil {
		return nil, fmt.Errorf("ошибка при получении доставок: %w", err)
	}
	deliveryMap := make(map[string]*model.Delivery)
	for _, d := range deliveries {
		deliveryMap[d.OrderUID] = d
	}

	var payments []*model.Payment
	paymentQuery := `SELECT * FROM payments`
	if err := repo.SelectContext(ctx, &payments, paymentQuery); err != nil {
		return nil, fmt.Errorf("ошибка при получении оплат: %w", err)
	}
	paymentMap := make(map[string]*model.Payment)
	for _, p := range payments {
		paymentMap[p.OrderUID] = p
	}

	var items []*model.Item
	itemsQuery := `SELECT * FROM items`
	if err := repo.SelectContext(ctx, &items, itemsQuery); err != nil {
		return nil, fmt.Errorf("ошибка при получении items: %w", err)
	}
	itemsMap := make(map[string][]model.Item)
	for _, item := range items {
		itemsMap[item.OrderUID] = append(itemsMap[item.OrderUID], *item)
	}

	var fullOrders []model.FullOrder
	for _, order := range orders {
		full := model.FullOrder{
			Order:    order,
			Delivery: deliveryMap[order.OrderUID],
			Payment:  paymentMap[order.OrderUID],
			Items:    itemsMap[order.OrderUID],
		}
		fullOrders = append(fullOrders, full)
	}

	return fullOrders, nil
}

func (repo *OrderRepository) SaveFullOrderTx(ctx context.Context, order *model.FullOrder) error {
	transaction, err := repo.BeginTxx(ctx, nil)
	if err != nil {
		return util.LogError("не удалось начать транзакцию", err)
	}
	defer func() {
		if r := recover(); r != nil {
			transaction.Rollback()
			panic(r)
		}
	}()

	if err := repo.saveOrder(ctx, transaction, order.Order); err != nil {
		transaction.Rollback()
		return util.LogError("не удалось выполнить транзакцию для заказа", err)
	}
	if err := repo.deliveryRepo.SaveDelivery(ctx, transaction, order.Delivery); err != nil {
		transaction.Rollback()
		return util.LogError("не удалось выполнить транзакцию для доставки", err)
	}
	if err := repo.paymentRepo.SavePayment(ctx, transaction, order.Payment); err != nil {
		transaction.Rollback()
		return util.LogError("не удалось выполнить транзакцию для оплаты", err)
	}
	if err := repo.itemsRepo.SaveItems(ctx, transaction, order.Items); err != nil {
		transaction.Rollback()
		return util.LogError("не удалось выполнить транзакцию для товара(ов)", err)
	}

	log.Printf("заказ от %s с uuid=%s успешно сохранен", order.Order.DateCreated, order.Order.OrderUID)

	return transaction.Commit()
}

func (repo *OrderRepository) UpdateFullOrderTx(ctx context.Context, order *model.FullOrder) error {
	transaction, err := repo.BeginTxx(ctx, nil)
	if err != nil {
		return util.LogError("не удалось начать транзакцию", err)
	}
	defer func() {
		if r := recover(); r != nil {
			transaction.Rollback()
			panic(r)
		}
	}()

	if err := repo.updateOrder(ctx, transaction, order.Order); err != nil {
		transaction.Rollback()
		return util.LogError("не удалось выполнить транзакцию для заказа", err)
	}
	if err := repo.deliveryRepo.UpdateDelivery(ctx, transaction, order.Delivery); err != nil {
		transaction.Rollback()
		return util.LogError("не удалось выполнить транзакцию для доставки", err)
	}
	if err := repo.paymentRepo.UpdatePayment(ctx, transaction, order.Payment); err != nil {
		transaction.Rollback()
		return util.LogError("не удалось выполнить транзакцию для оплаты", err)
	}

	log.Printf("заказ от %s с uuid=%s успешно обновлен", order.Order.DateCreated, order.Order.OrderUID)

	return transaction.Commit()
}

func (repo *OrderRepository) DeleteOrderByUUIDTx(ctx context.Context, uuid string) error {
	transaction, err := repo.BeginTxx(ctx, nil)
	if err != nil {
		return util.LogError("не удалось начать транзакцию", err)
	}
	defer func() {
		if r := recover(); r != nil {
			transaction.Rollback()
			panic(r)
		}
	}()

	query := `DELETE FROM orders WHERE order_uid = $1`
	_, err = transaction.ExecContext(ctx, query, uuid)
	if err != nil {
		return util.LogError("ошибка удаления заказа", err)
	}

	log.Printf("заказ с uuid=%s успешно удален", uuid)

	return transaction.Commit()
}

func (repo *OrderRepository) GetOrderByUUID(ctx context.Context, exec sqlx.ExtContext, uuid string) (*model.Order, error) {
	query := `SELECT * FROM orders WHERE order_uid=$1`

	var returnedOrder model.Order
	err := sqlx.GetContext(ctx, exec, &returnedOrder, query, uuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.LogError("не удалось вставить данные в таблицу", err)
		}
		return nil, util.LogError("ошибка получения таблицы заказов", err)
	}

	return &returnedOrder, nil
}

func (repo *OrderRepository) saveOrder(ctx context.Context, exec sqlx.ExtContext, order *model.Order) error {
	query := `INSERT INTO orders 
    (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := exec.ExecContext(
		ctx,
		query,
		order.OrderUID,
		order.TrackNumber,
		order.Entry,
		order.Locale,
		order.InternalSignature,
		order.CustomerID,
		order.DeliveryService,
		order.ShardKey,
		order.SmID,
		order.DateCreated,
		order.OofShard,
	)

	if err != nil {
		return util.LogError("ошибка при вставке заказа", err)
	}

	return nil
}

func (repo *OrderRepository) updateOrder(ctx context.Context, exec sqlx.ExtContext, order *model.Order) error {
	query := `UPDATE orders
		SET track_number = $1,
		    entry = $2,
		    delivery_service = $3,
		    sm_id = $4
		WHERE order_uid = $5`

	res, err := exec.ExecContext(ctx, query,
		order.TrackNumber,
		order.Entry,
		order.DeliveryService,
		order.SmID,
		order.OrderUID,
	)
	if err != nil {
		return util.LogError("ошибка при обновлении заказа", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return util.LogError("не удалось получить количество затронутых строк", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("обновление заказа: заказ с order_uid=%s не найден", order.OrderUID)
	}

	return nil
}
