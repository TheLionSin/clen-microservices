package postgresrepo

import (
	"context"
	"errors"
	"fmt"
	"user-service/internal/domain"
	"user-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) repository.UserRepository {
	return &userRepo{
		pool: pool,
	}
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
			INSERT INTO users (id,email,password_hash,role,created_at) VALUES($1,$2,$3,$4,$5)`

	_, err := r.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt)
	if err != nil {
		// Проверяем, является ли ошибка нарушением уникальности (Unique Violation)
		// Код 23505 в PostgreSQL означает "unique_violation"
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrUserAlreadyExists
		}
		return fmt.Errorf("postgresrepo.userRepo.Create: %w", err)
	}

	return nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
			SELECT id,email,password_hash,role,created_at FROM users WHERE email = $1`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)

	if err != nil {
		// Перехватываем отсутствие строк и возвращаем красивую доменную ошибку
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("postgresrepo.userRepo.GetByEmail: %w", err)
	}

	return &user, nil
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
			SELECT id,email,password_hash,role,created_at FROM users WHERE id = $1`

	var user domain.User

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("postgresrepo.userRepo.GetByID: %w", err)
	}

	return &user, nil
}

func (r *userRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error {
	query := `
			UPDATE users SET password_hash = $1 WHERE id = $2`

	// Exec возвращает pgconn.CommandTag, из которого можно узнать, сколько строк было изменено
	cmdTag, err := r.pool.Exec(ctx, query, newPasswordHash, userID)
	if err != nil {
		return fmt.Errorf("postgresrepo.UpdatePassword: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}
