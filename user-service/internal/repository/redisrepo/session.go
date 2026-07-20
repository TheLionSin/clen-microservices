package redisrepo

import (
	"context"
	"errors"
	"fmt"
	"time"
	"user-service/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ErrSessionNotFound = errors.New("session not found")

type sessionRepo struct {
	client *redis.Client
}

func NewSessionRepo(client *redis.Client) repository.SessionRepository {
	return &sessionRepo{client: client}
}

func (r *sessionRepo) SetRefreshToken(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	// Ключ: "refresh:<token>", Значение: userID.
	// Используем встроенный TTL Редиса, чтобы токены удалялись сами через 30 дней.
	key := fmt.Sprintf("refresh:%s", token)
	err := r.client.Set(ctx, key, userID.String(), ttl).Err()
	if err != nil {
		return fmt.Errorf("redisrepo.SetRefreshToken: %w", err)
	}
	return nil
}

func (r *sessionRepo) GetUserIDByToken(ctx context.Context, token string) (uuid.UUID, error) {
	key := fmt.Sprintf("refresh:%s", token)
	userIDStr, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Токен истек (удалился по TTL) или юзер уже сделал Logout
			return uuid.Nil, ErrSessionNotFound
		}
		return uuid.Nil, fmt.Errorf("redisrepo.GetUserIDByToken: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("redisrepo.GetUserIDByToken parse uuid: %w", err)
	}

	return userID, nil
}

func (r *sessionRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("refresh:%s", token)
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redisrepo.DeleteRefreshToken: %w", err)
	}
	return nil
}
