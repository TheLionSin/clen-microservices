package repository

import (
	"catalog-service/internal/domain"
	"context"

	"github.com/google/uuid"
)

type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	List(ctx context.Context, limit, offset int) ([]domain.Product, error)
}
