package usecase

import (
	"catalog-service/internal/domain"
	"catalog-service/internal/repository"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateProductInput struct {
	CategoryID  uuid.UUID
	Name        string
	Description string
	Price       int64
	Stock       int
	ImageURLs   []string
}

type UpdateProductInput struct {
	ID          uuid.UUID
	CategoryID  uuid.UUID
	Name        string
	Description string
	Price       int64
	Stock       int
	ImageURLs   []string
}

type OrderItemInput struct {
	ProductID uuid.UUID
	Quantity  int
}

type ProductUseCase interface {
	Create(ctx context.Context, input CreateProductInput) (uuid.UUID, error)
	Update(ctx context.Context, input UpdateProductInput) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	List(ctx context.Context, limit, offset int) ([]domain.Product, error)
	ProcessOrderCreated(ctx context.Context, items []OrderItemInput) error
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
	if input.CategoryID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("%w: category_id is required", ErrInvalidInput)
	}
	if input.ImageURLs == nil {
		input.ImageURLs = make([]string, 0)
	}

	now := time.Now().UTC()
	newProduct := &domain.Product{
		ID:          uuid.New(),
		CategoryID:  input.CategoryID,
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		Stock:       input.Stock,
		ImageURLs:   input.ImageURLs,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := u.repo.Create(ctx, newProduct); err != nil {
		return uuid.Nil, fmt.Errorf("usecase.product.Create: %w", err)
	}

	return newProduct.ID, nil
}

func (u *productUseCase) Update(ctx context.Context, input UpdateProductInput) error {
	if input.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if input.Price <= 0 {
		return fmt.Errorf("%w: price must be greater than zero", ErrInvalidInput)
	}
	if input.Stock < 0 {
		return fmt.Errorf("%w: stock cannot be negative", ErrInvalidInput)
	}

	productToUpdate := &domain.Product{
		ID:          input.ID,
		CategoryID:  input.CategoryID,
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		Stock:       input.Stock,
		ImageURLs:   input.ImageURLs,
		UpdatedAt:   time.Now().UTC(),
	}

	if err := u.repo.Update(ctx, productToUpdate); err != nil {
		return fmt.Errorf("usecase.product.Update: %w", err)
	}
	return nil
}

func (u *productUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := u.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("usecase.product.Delete: %w", err)
	}
	return nil
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

func (u *productUseCase) ProcessOrderCreated(ctx context.Context, items []OrderItemInput) error {
	for _, item := range items {
		if item.Quantity <= 0 {
			continue
		}

		err := u.repo.DecrementStock(ctx, item.ProductID, item.Quantity)
		if err != nil {
			return fmt.Errorf("usecase.product.ProcessOrderCreated (product_id: %s): %w", item.ProductID, err)
		}
	}

	return nil
}
