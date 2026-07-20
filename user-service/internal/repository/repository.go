package repository

import (
	"context"
	"time"
	"user-service/internal/domain"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error
}

// SessionRepository управляет Refresh токенами (сессиями)
type SessionRepository interface {
	SetRefreshToken(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error
	GetUserIDByToken(ctx context.Context, token string) (uuid.UUID, error)
	DeleteRefreshToken(ctx context.Context, token string) error
}
