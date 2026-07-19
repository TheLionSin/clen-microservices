package repository

import (
	"context"
	"order-service/internal/domain"

	"github.com/google/uuid"
)

type CartRepository interface {
	Get(ctx context.Context, userID uuid.UUID) (*domain.Cart, error)
	Save(ctx context.Context, cart *domain.Cart) error
	Delete(ctx context.Context, userID uuid.UUID) error
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *domain.Order) error
	GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
}
