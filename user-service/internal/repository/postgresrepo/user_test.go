package postgresrepo

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
	"user-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(10*time.Second)))
	require.NoError(t, err, "failed to start postgres container")

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get database connection string")

	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)
	defer db.Close()

	migrationsDir, err := filepath.Abs("../../../migrations")
	require.NoError(t, err)

	err = goose.SetDialect("postgres")
	require.NoError(t, err)
	err = goose.Up(db, migrationsDir)
	require.NoError(t, err, "failed to run migrations")

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	cleanup := func() {
		pool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}

	return pool, cleanup
}

func TestUserRepository_Integration(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepo(pool)
	ctx := context.Background()

	userID := uuid.New()
	email := "integration@mail.ru"
	passwordHash := "initial_hash_123"

	userToCreate := &domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         "user",
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}

	err := repo.Create(ctx, userToCreate)
	require.NoError(t, err)

	duplicateUser := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: "some_hash",
		Role:         "user",
		CreatedAt:    time.Now().UTC(),
	}
	err = repo.Create(ctx, duplicateUser)
	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)

	foundByEmail, err := repo.GetByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, userID, foundByEmail.ID)
	assert.Equal(t, passwordHash, foundByEmail.PasswordHash)

	foundByID, err := repo.GetByID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, email, foundByID.Email)

	_, err = repo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrUserNotFound)

	newHash := "new_hash_456"
	err = repo.UpdatePassword(ctx, userID, newHash)
	require.NoError(t, err)

	updatedUser, err := repo.GetByID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, newHash, updatedUser.PasswordHash)
}
