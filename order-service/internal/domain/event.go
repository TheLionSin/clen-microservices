package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderItemEvent struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

// OrderCreatedEvent - структура, которая будет сериализована в JSON и отправлена в Kafka
type OrderCreatedEvent struct {
	OrderID     uuid.UUID        `json:"order_id"`
	UserID      uuid.UUID        `json:"user_id"`
	TotalAmount int64            `json:"total_amount"`
	Items       []OrderItemEvent `json:"items"`
	CreatedAt   time.Time        `json:"created_at"`
}
