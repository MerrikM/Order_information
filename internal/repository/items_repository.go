package repository

import (
	"Order_information/internal/config"
	"Order_information/internal/model"
	"Order_information/util"
	"context"
	"github.com/jmoiron/sqlx"
)

type ItemsRepository struct {
	*config.Database
}

func NewItemsRepository(database *config.Database) *ItemsRepository {
	return &ItemsRepository{database}
}

func (repo *ItemsRepository) GetItemsByOrderUID(ctx context.Context, exec sqlx.ExtContext, orderUID string) ([]model.Item, error) {
	query := `SELECT * FROM items WHERE order_uid=$1`

	var items []model.Item
	err := sqlx.SelectContext(ctx, exec, &items, query, orderUID)
	if err != nil {
		return nil, util.LogError("ошибка при получении таблицы товаров", err)
	}

	return items, nil
}

func (repo *ItemsRepository) SaveItem(ctx context.Context, exec sqlx.ExtContext, item *model.Item, orderUID string) error {
	query := `INSERT INTO items 
    (chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status, order_uid) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := exec.ExecContext(
		ctx,
		query,
		item.ChrtID,
		item.TrackNumber,
		item.Price,
		item.RID,
		item.Name,
		item.Sale,
		item.Size,
		item.TotalPrice,
		item.NmID,
		item.Brand,
		item.Status,
		orderUID,
	)

	if err != nil {
		return util.LogError("ошибка при вставке товара", err)
	}

	return nil
}

func (repo *ItemsRepository) SaveItems(ctx context.Context, exec sqlx.ExtContext, items []model.Item, orderUID string) error {
	for _, item := range items {
		if err := repo.SaveItem(ctx, exec, &item, orderUID); err != nil {
			return err
		}
	}
	return nil
}
