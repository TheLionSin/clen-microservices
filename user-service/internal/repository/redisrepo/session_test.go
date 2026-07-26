package redisrepo

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	ctx := context.Background()

	redisContainer, err := testredis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err, "failed to start redis container")

	uri, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	opts, err := redis.ParseURL(uri)
	require.NoError(t, err)
	client := redis.NewClient(opts)

	err = client.Ping(ctx).Err()
	require.NoError(t, err, "failed to ping redis")

	cleanup := func() {
		client.Close()
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate redis container: %s", err)
		}
	}

	return client, cleanup
}

func TestSessionRepository_Integration(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewSessionRepo(client)
	ctx := context.Background()

	userID := uuid.New()
	token := "test_refresh_token_123"
	ttl := 1 * time.Hour

	//1. Пытаемся получить несуществующий токен
	_, err := repo.GetUserIDByToken(ctx, "unknown_token")
	assert.ErrorIs(t, err, ErrSessionNotFound)

	//2. Сохраняем новый токен
	err = repo.SetRefreshToken(ctx, token, userID, ttl)
	require.NoError(t, err)

	//3. Получаем сохраненный токен
	savedUserID, err := repo.GetUserIDByToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, userID, savedUserID)

	//4. Удаляем токен (имитация Логаут или Ротации)
	err = repo.DeleteRefreshToken(ctx, token)
	require.NoError(t, err)

	_, err = repo.GetUserIDByToken(ctx, token)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}
