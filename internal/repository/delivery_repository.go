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

func (repo *DeliveryRepository) GetDeliveryById(ctx context.Context, id string) (*model.Delivery, error) {
	query := `SELECT * FROM items WHERE order_uid=$1`

	var returnedDelivery model.Delivery
	err := repo.GetContext(ctx, &returnedDelivery, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.LogError("не удалось вставить данные в таблицу", err)
		}
		return nil, util.LogError("ошибка получения таблицы", err)
	}

	return &returnedDelivery, nil
}

func (repo *DeliveryRepository) SaveDelivery(ctx context.Context, exec sqlx.ExtContext, delivery *model.Delivery) error {
	query := `INSERT INTO deliveries 
    (id, order_uid, name, phone, zip, city, address, region, email) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := exec.ExecContext(
		ctx,
		query,
		delivery.ID,
		delivery.OrderUID,
		delivery.Name,
		delivery.Phone,
		delivery.Zip,
		delivery.City,
		delivery.Address,
		delivery.Region,
		delivery.Email,
	)

	if err != nil {
		return util.LogError("ошибка при вставке информации о доставке", err)
	}

	return nil
}

