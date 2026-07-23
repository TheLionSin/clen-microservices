package postgresrepo

import (
	"context"
	"database/sql"
	"order-service/internal/domain"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB поднимает контейнер с PostgreSQL и накатывает миграции.
// Возвращает пул соединений и функцию для очистки (удаления контейнера).
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()

	//1. Даем команду докеру поднять контейнер с Postgres 16
	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("test_db"), postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(10*time.Second)))
	require.NoError(t, err)

	//2. Получаем строку подключения к временному контейнеру
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	//3. Подключаемся через стандартный sql.DB для накатывания миграций(goose требует sql.DB)
	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)
	defer db.Close()

	// Указываем путь к нашим файлам миграций (поднимаемся на уровень выше до папки migrations)
	migrationsDir, err := filepath.Abs("../../../migrations")
	require.NoError(t, err)

	//Накатываем миграции
	err = goose.SetDialect("postgres")
	require.NoError(t, err)
	err = goose.Up(db, migrationsDir)
	require.NoError(t, err, "failed to run migrations")

	//4. Создаем боевой pgxpool, который использует наш репозиторий
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	//Возвращаем пул и функцию, которая убьет контейнер после теста
	cleanup := func() {
		pool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}

	return pool, cleanup
}

func TestCreateOrder_Integration(t *testing.T) {
	//Поднимаем бд и гарантируем ее удаление после теста
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	//Создаем репозиторий передавая ему пул соединений от тестового контейнера
	repo := NewOrderRepo(pool)
	ctx := context.Background()

	//Тестовые данные
	orderID := uuid.New()
	userID := uuid.New()

	newOrder := &domain.Order{
		ID:          orderID,
		UserID:      userID,
		TotalAmount: 15000,
		Status:      "pending",
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond), // База режет точность до микросекунд
		Items: []domain.OrderItem{
			{
				ID:        uuid.New(),
				OrderID:   orderID,
				ProductID: uuid.New(),
				Quantity:  3,
				Price:     5000,
			},
		},
	}

	//Пытаемся сохранить заказ
	err := repo.CreateOrder(ctx, newOrder)
	require.NoError(t, err)

	// Дополнительная проверка: идем сырым SQL-запросом в базу и проверяем,
	// что запись реально появилась в таблице orders
	var savedTotal int64
	var savedStatus string
	err = pool.QueryRow(ctx, "SELECT total_amount, status FROM orders WHERE id = $1", orderID).Scan(&savedTotal, &savedStatus)

	assert.NoError(t, err)
	assert.Equal(t, int64(15000), savedTotal)
	assert.Equal(t, "pending", savedStatus)
}

func TestGetOrdersByUserID_Integration(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepo(pool)
	ctx := context.Background()

	targetUserID := uuid.New()
	otherUserID := uuid.New()

	order1 := &domain.Order{
		ID:          uuid.New(),
		UserID:      targetUserID,
		TotalAmount: 5000,
		Status:      "paid",
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		Items: []domain.OrderItem{
			{ID: uuid.New(), ProductID: uuid.New(), Quantity: 1, Price: 5000},
		},
	}
	order1.Items[0].OrderID = order1.ID
	require.NoError(t, repo.CreateOrder(ctx, order1))

	order2 := &domain.Order{
		ID:          uuid.New(),
		UserID:      targetUserID,
		TotalAmount: 10000,
		Status:      "pending",
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		Items: []domain.OrderItem{
			{ID: uuid.New(), ProductID: uuid.New(), Quantity: 2, Price: 5000},
		},
	}
	order2.Items[0].OrderID = order2.ID
	require.NoError(t, repo.CreateOrder(ctx, order2))

	order3 := &domain.Order{
		ID:          uuid.New(),
		UserID:      otherUserID,
		TotalAmount: 15000,
		Status:      "pending",
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		Items: []domain.OrderItem{
			{ID: uuid.New(), ProductID: uuid.New(), Quantity: 3, Price: 5000},
		},
	}
	order3.Items[0].OrderID = order3.ID
	require.NoError(t, repo.CreateOrder(ctx, order3))

	//Запрашиваем заказы целевого юзера
	orders, err := repo.GetOrdersByUserID(ctx, targetUserID)
	assert.NoError(t, err)

	//Ожидаем ровно 2 заказа (заказ другого юзера не должен попасть в выборку)
	assert.Len(t, orders, 2)

	// Дополнительно проверяем, что вернулись именно те суммы, которые мы ожидаем
	var totalSums []int64
	for _, o := range orders {
		totalSums = append(totalSums, o.TotalAmount)
	}
	assert.Contains(t, totalSums, int64(5000))
	assert.Contains(t, totalSums, int64(10000))
}
