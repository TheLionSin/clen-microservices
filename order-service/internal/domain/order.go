package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusCreated OrderStatus = "created"
	StatusPaid    OrderStatus = "paid"
	StatusFailed  OrderStatus = "failed"
)

// OrderItem - конкретный товар внутри оформленного заказа
type OrderItem struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Quantity  int
	Price     int64
}

// Order - сам заказ
type Order struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	TotalAmount int64
	Status      OrderStatus
	Items       []OrderItem // Связь один-ко-многим (один заказ - много позиций)
	CreatedAt   time.Time
}
