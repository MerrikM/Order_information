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

type PaymentRepository struct {
	*config.Database
}

func NewPaymentRepository(database *config.Database) *PaymentRepository {
	return &PaymentRepository{database}
}

func (repo *PaymentRepository) GetPaymentByOrderUID(ctx context.Context, exec sqlx.ExtContext, id string) (*model.Payment, error) {
	query := `SELECT * FROM payments WHERE order_uid=$1`

	var returnedPayment model.Payment
	err := sqlx.GetContext(ctx, exec, &returnedPayment, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.LogError("не удалось вставить данные в таблицу", err)
		}
		return nil, util.LogError("ошибка получения таблицы с информацией об оплате", err)
	}

	return &returnedPayment, nil
}

func (repo *PaymentRepository) SavePayment(ctx context.Context, exec sqlx.ExtContext, payment *model.Payment, orderUID string) error {
	query := `INSERT INTO payments 
    (transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee, order_uid) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := exec.ExecContext(
		ctx,
		query,
		payment.Transaction,
		payment.RequestID,
		payment.Currency,
		payment.Provider,
		payment.Amount,
		payment.PaymentDT,
		payment.Bank,
		payment.DeliveryCost,
		payment.GoodsTotal,
		payment.CustomFee,
		orderUID,
	)

	if err != nil {
		return util.LogError("ошибка при вставке информации об оплате", err)
	}

	return nil
}

func (repo *PaymentRepository) UpdatePayment(ctx context.Context, exec sqlx.ExtContext, payment *model.Payment) error {
	query := `UPDATE payments
		SET request_id = $1,
		    currency = $2,
		    amount = $3,
		    payment_dt = $4,
		    bank = $5,
		    delivery_cost = $6,
		    custom_fee = $7
		WHERE transaction = $8`

	res, err := exec.ExecContext(ctx, query,
		payment.RequestID,
		payment.Currency,
		payment.Amount,
		payment.PaymentDT,
		payment.Bank,
		payment.DeliveryCost,
		payment.CustomFee,
		payment.Transaction,
	)
	if err != nil {
		return util.LogError("ошибка при обновлении способа оплаты", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return util.LogError("не удалось получить количество затронутых строк", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("оплата с transaction=%s не найден", payment.Transaction)
	}

	return nil
}
