package event

import (
	"catalog-service/internal/usecase"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type OrderItemEvent struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

type OrderCreatedEvent struct {
	OrderID     uuid.UUID        `json:"order_id"`
	UserID      uuid.UUID        `json:"user_id"`
	TotalAmount int64            `json:"total_amount"`
	Items       []OrderItemEvent `json:"items"`
	CreatedAt   time.Time        `json:"created_at"`
}

type OrderConsumer struct {
	reader         *kafka.Reader
	productUseCase usecase.ProductUseCase
}

func NewOrderConsumer(brokers []string, topic, groupID string, productUseCase usecase.ProductUseCase) *OrderConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MaxBytes: 10e6,
	})

	return &OrderConsumer{
		reader:         reader,
		productUseCase: productUseCase,
	}
}

func (c *OrderConsumer) Start(ctx context.Context) {
	slog.Info("Starting Kafka Order Consumer in Catalog Service...")

	go func() {
		for {
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("Catalog Consumer: error reading message", slog.String("error", err.Error()))
				continue
			}

			var event OrderCreatedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				slog.Error("Catalog Consumer: error unmarshalling event", slog.String("error", err.Error()))
				continue
			}

			// Трансформируем ивент в инпут для Usecase
			items := make([]usecase.OrderItemInput, 0, len(event.Items))
			for _, item := range event.Items {
				items = append(items, usecase.OrderItemInput{
					ProductID: item.ProductID,
					Quantity:  item.Quantity,
				})
			}

			// Списываем остатки
			if err := c.productUseCase.ProcessOrderCreated(ctx, items); err != nil {
				slog.Error("Catalog Consumer: failed to process order stock decrement",
					slog.String("order_id", event.OrderID.String()),
					slog.String("error", err.Error()))
				continue
			}
			slog.Info("Catalog Consumer: stock successfully updated for order", slog.String("order_id", event.OrderID.String()))
		}
	}()
}

func (c *OrderConsumer) Close() error {
	return c.reader.Close()
}
