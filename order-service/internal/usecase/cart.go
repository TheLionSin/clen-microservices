package usecase

import (
	"context"
	"errors"
	"fmt"
	"order-service/internal/domain"
	"order-service/internal/repository"
	grpcclient "order-service/pkg/client/grpc"

	"github.com/google/uuid"
)

var (
	ErrInvalidQuantity = errors.New("quantity must be greater than zero")
	ErrProductNotFound = errors.New("product not found in catalog")
	ErrNotEnoughStock  = errors.New("not enough stock in catalog")
)

type AddToCartInput struct {
	UserID    uuid.UUID
	ProductID uuid.UUID
	Quantity  int
}

type CartUseCase interface {
	AddToCart(ctx context.Context, input AddToCartInput) (*domain.Cart, error)
	GetCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error)
	ClearCart(ctx context.Context, userID uuid.UUID) error
}

type cartUseCase struct {
	repo          repository.CartRepository
	catalogClient *grpcclient.CatalogClient
}

func NewCartUseCase(repo repository.CartRepository, catalogClient *grpcclient.CatalogClient) CartUseCase {
	return &cartUseCase{
		repo:          repo,
		catalogClient: catalogClient,
	}
}

func (u *cartUseCase) AddToCart(ctx context.Context, input AddToCartInput) (*domain.Cart, error) {
	if input.Quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	resp, err := u.catalogClient.CheckProduct(ctx, input.ProductID.String())
	if err != nil {
		return nil, fmt.Errorf("usecase.AddToCart catalog check: %w", err)
	}

	if !resp.Exists {
		return nil, ErrProductNotFound
	}
	if resp.Stock < int32(input.Quantity) {
		return nil, ErrNotEnoughStock
	}

	cart, err := u.repo.Get(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrCartNotFound) {
			cart = &domain.Cart{
				UserID: input.UserID,
				Items:  make([]domain.CartItem, 0),
			}
		} else {
			return nil, fmt.Errorf("usecase.AddToCart get cart: %w", err)
		}
	}

	itemExists := false
	for i, item := range cart.Items {
		if item.ProductID == input.ProductID {
			// Товар уже есть в корзине, просто увеличиваем количество
			cart.Items[i].Quantity += input.Quantity
			itemExists = true
			break
		}
	}

	if !itemExists {
		cart.Items = append(cart.Items, domain.CartItem{
			ProductID: input.ProductID,
			Quantity:  input.Quantity,
		})
	}

	// Сохраняем обновленную корзину обратно в Redis
	if err := u.repo.Save(ctx, cart); err != nil {
		return nil, fmt.Errorf("usecase.AddToCart save cart: %w", err)
	}

	return cart, nil
}

func (u *cartUseCase) GetCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	cart, err := u.repo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase.GetCart: %w", err)
	}
	return cart, nil
}

func (u *cartUseCase) ClearCart(ctx context.Context, userID uuid.UUID) error {
	if err := u.repo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("usecase.ClearCart: %w", err)
	}
	return nil
}
