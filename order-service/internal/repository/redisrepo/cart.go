package redisrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"order-service/internal/domain"
	"order-service/internal/repository"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	cartPrefix = "cart:"
	cartTTL    = 7 * 24 * time.Hour
)

type cartRepo struct {
	client *redis.Client
}

func NewCartRepo(client *redis.Client) repository.CartRepository {
	return &cartRepo{client: client}
}

func (r *cartRepo) Get(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	key := cartPrefix + userID.String()

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrCartNotFound
		}
		return nil, fmt.Errorf("redisrepo.cartRepo.Get: %w", err)
	}

	var cart domain.Cart
	if err := json.Unmarshal([]byte(val), &cart); err != nil {
		return nil, fmt.Errorf("redisrepo.cartRepo.Get unmarshal: %w", err)
	}

	return &cart, nil
}

func (r *cartRepo) Save(ctx context.Context, cart *domain.Cart) error {
	key := cartPrefix + cart.UserID.String()

	data, err := json.Marshal(cart)
	if err != nil {
		return fmt.Errorf("redisrepo.cartRepo.Save marshal: %w", err)
	}

	if err := r.client.Set(ctx, key, data, cartTTL).Err(); err != nil {
		return fmt.Errorf("redisrepo.cartRepo.Save: %w", err)
	}

	return nil
}

func (r *cartRepo) Delete(ctx context.Context, userID uuid.UUID) error {
	key := cartPrefix + userID.String()

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redisrepo.cartRepo.Delete: %w", err)
	}

	return nil
}
