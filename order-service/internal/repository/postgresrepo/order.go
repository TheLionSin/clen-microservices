package postgresrepo

import (
	"context"
	"fmt"
	"order-service/internal/domain"
	"order-service/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type orderRepo struct {
	pool *pgxpool.Pool
}

func NewOrderRepo(pool *pgxpool.Pool) repository.OrderRepository {
	return &orderRepo{pool: pool}
}

func (r *orderRepo) CreateOrder(ctx context.Context, order *domain.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка старта транзакции: %w", err)
	}

	// Обязательный паттерн: defer Rollback.
	// Если мы не вызовем tx.Commit() из-за какой-то ошибки ниже,
	// defer отменит все изменения. Если Commit() прошел успешно, Rollback просто ничего не сделает.
	defer tx.Rollback(ctx)

	orderQuery := `
			INSERT INTO orders (id, user_id, total_amount, status, created_at)
			VALUES ($1, $2, $3, $4, $5)`

	_, err = tx.Exec(ctx, orderQuery, order.ID, order.UserID, order.TotalAmount, order.Status, order.CreatedAt)
	if err != nil {
		return fmt.Errorf("ошибка сохранения заказа: %w", err)
	}

	itemQuery := `
			INSERT INTO order_items (id, order_id, product_id, quantity, price)
			VALUES ($1, $2, $3, $4, $5)`

	for _, item := range order.Items {
		_, err = tx.Exec(ctx, itemQuery, item.ID, item.OrderID, item.ProductID, item.Quantity, item.Price)
		if err != nil {
			return fmt.Errorf("ошибка сохранения позиции заказа: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}
