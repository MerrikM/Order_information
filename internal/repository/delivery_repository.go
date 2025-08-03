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

type DeliveryRepository struct {
	*config.Database
}

func NewDeliveryRepository(database *config.Database) *DeliveryRepository {
	return &DeliveryRepository{database}
}

func (repo *DeliveryRepository) GetDeliveryByOrderUID(ctx context.Context, exec sqlx.ExtContext, uuid string) (*model.Delivery, error) {
	query := `SELECT * FROM deliveries WHERE order_uid=$1`

	var returnedDelivery model.Delivery
	err := sqlx.GetContext(ctx, exec, &returnedDelivery, query, uuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.LogError("не удалось вставить данные в таблицу", err)
		}
		return nil, util.LogError("ошибка получения таблицы доставки", err)
	}

	return &returnedDelivery, nil
}

func (repo *DeliveryRepository) SaveDelivery(ctx context.Context, exec sqlx.ExtContext, delivery *model.Delivery) error {
	query := `INSERT INTO deliveries 
    (order_uid, name, phone, zip, city, address, region, email) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id
	`
	err := exec.QueryRowxContext(
		ctx,
		query,
		delivery.OrderUID,
		delivery.Name,
		delivery.Phone,
		delivery.Zip,
		delivery.City,
		delivery.Address,
		delivery.Region,
		delivery.Email,
	).Scan(&delivery.ID)

	if err != nil {
		return util.LogError("ошибка при вставке информации о доставке", err)
	}

	return nil
}

func (repo *DeliveryRepository) UpdateDelivery(ctx context.Context, exec sqlx.ExtContext, delivery *model.Delivery) error {
	query := `UPDATE deliveries
		SET name= $1,
		    phone = $2,
		    zip = $3,
		    city = $4,
		    address = $5,
		    region = $6,
		    email = $7
		WHERE order_uid = $8`

	res, err := exec.ExecContext(ctx, query,
		delivery.Name,
		delivery.Phone,
		delivery.Zip,
		delivery.City,
		delivery.Address,
		delivery.Region,
		delivery.Email,
		delivery.OrderUID,
	)
	if err != nil {
		return util.LogError("ошибка при обновлении информации о доставке", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return util.LogError("не удалось получить количество затронутых строк", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("обновление информации о доставке: доставка с id=%d не найдена", delivery.ID)
	}

	return nil
}
