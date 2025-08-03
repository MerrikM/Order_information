package ports

import (
	"Order_information/internal/model"
	"context"
)

type Cache interface {
	SetOrder(ctx context.Context, order *model.FullOrder) error
	GetOrder(ctx context.Context, uuid string) (*model.FullOrder, error)
	DeleteOrder(ctx context.Context, uuid string) error
}
