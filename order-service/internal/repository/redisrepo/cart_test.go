package redisrepo

import (
	"context"
	"order-service/internal/domain"
	"testing"

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

func TestCartRepository_Integration(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewCartRepo(client)
	ctx := context.Background()

	userID := uuid.New()
	productID := uuid.New()

	//1.Проверяем ошибку при пустой корзине
	_, err := repo.Get(ctx, userID)
	assert.ErrorIs(t, err, domain.ErrCartNotFound)

	//2.Сохраняем новую корзину
	cartToSave := &domain.Cart{
		UserID: userID,
		Items: []domain.CartItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	err = repo.Save(ctx, cartToSave)
	require.NoError(t, err)

	//3.Получаем сохраненную корзину и сверяем данные

	savedCart, err := repo.Get(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, savedCart)
	assert.Equal(t, userID, savedCart.UserID)
	assert.Len(t, savedCart.Items, 1)
	assert.Equal(t, productID, savedCart.Items[0].ProductID)
	assert.Equal(t, 2, savedCart.Items[0].Quantity)

	//4.Удаляем корзину
	err = repo.Delete(ctx, userID)
	require.NoError(t, err)

	_, err = repo.Get(ctx, userID)
	assert.ErrorIs(t, err, domain.ErrCartNotFound)
}
