package usecase

import (
	"catalog-service/internal/domain"
	"catalog-service/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput = errors.New("invalid input data")
)

type CreateProductInput struct {
	Name        string
	Description string
	Price       int64
	Stock       int
}

type ProductUseCase interface {
	Create(ctx context.Context, input CreateProductInput) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	List(ctx context.Context, limit, offset int) ([]domain.Product, error)
}

type productUseCase struct {
	repo repository.ProductRepository
}

func NewProduct(repo repository.ProductRepository) ProductUseCase {
	return &productUseCase{
		repo: repo,
	}
}

func (u *productUseCase) Create(ctx context.Context, input CreateProductInput) (uuid.UUID, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return uuid.Nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if input.Price <= 0 {
		return uuid.Nil, fmt.Errorf("%w: price must be greater than zero", ErrInvalidInput)
	}
	if input.Stock < 0 {
		return uuid.Nil, fmt.Errorf("%w: stock cannot be negative", ErrInvalidInput)
	}

	now := time.Now().UTC()
	newProduct := &domain.Product{
		ID:          uuid.New(),
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		Stock:       input.Stock,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := u.repo.Create(ctx, newProduct); err != nil {
		return uuid.Nil, fmt.Errorf("usecase.product.Create: %w", err)
	}

	return newProduct.ID, nil
}

func (u *productUseCase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	p, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase.product.GetByID: %w", err)
	}
	return p, nil
}

func (u *productUseCase) List(ctx context.Context, limit, offset int) ([]domain.Product, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	products, err := u.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("usecase.product.List: %w", err)
	}
	return products, nil
}
