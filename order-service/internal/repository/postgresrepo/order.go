package postgresrepo

import (
	"context"
	"fmt"
	"order-service/internal/domain"
	"order-service/internal/repository"

	"github.com/google/uuid"
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

func (r *orderRepo) GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	//Достаем сами заказы (сортируем от новых к старым)
	orderQuery := `
		SELECT id, user_id, total_amount, status, created_at
		FROM orders WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, orderQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("postgresrepo.GetOrders query: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	var orderIDs []uuid.UUID

	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.TotalAmount, &o.Status, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgresrepo.GetOrders scan: %w", err)
		}
		//Инициализируем пустой срез, чтобы в JSON отдавался [], а не null
		o.Items = []domain.OrderItem{}
		orders = append(orders, o)
		orderIDs = append(orderIDs, o.ID)
	}

	//Если заказов нет, просто возвращаем пустой массив
	if len(orders) == 0 {
		return []domain.Order{}, nil
	}

	//Создаем мапу (словарь) указателей для быстрого доступа к заказам в памяти
	orderMap := make(map[uuid.UUID]*domain.Order)
	for i := range orders {
		orderMap[orders[i].ID] = &orders[i]
	}

	//Достаем все товары для ВСЕХ найденных заказов ОДНИМ запросом
	//$1 здесь примет массив UUID, а оператор ANY сработает как IN (...)
	itemQuery := `
			SELECT id, order_id, product_id, quantity, price
			FROM order_items
			WHERE order_id = ANY($1)`

	itemRows, err := r.pool.Query(ctx, itemQuery, orderIDs)
	if err != nil {
		return nil, fmt.Errorf("postgresrepo.GetOrders items query: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item domain.OrderItem
		if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, fmt.Errorf("postgresrepo.GetOrders item scan: %w", err)
		}

		//Прикрепляем товар к нужному заказу через мапу
		if order, exists := orderMap[item.OrderID]; exists {
			order.Items = append(order.Items, item)
		}
	}

	return orders, nil
}
