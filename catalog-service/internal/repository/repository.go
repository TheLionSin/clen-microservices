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

type CategoryRepository interface {
	Create(ctx context.Context, category *domain.Category) error
	Update(ctx context.Context, category *domain.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]domain.Category, error)
}
