package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"notification-service/internal/config"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// OrderCreatedEvent - структура, точно совпадающая с тем, что отправляет Order Service
type OrderCreatedEvent struct {
	OrderID     uuid.UUID `json:"order_id"`
	UserID      uuid.UUID `json:"user_id"`
	TotalAmount int64     `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
}

func main() {
	cfg := config.GetConfig()
	setupLogger(cfg.IsDebug)
	slog.Info("Starting Notification Service")

	//1.Kafka Reader(слушатель)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Brokers,
		Topic:    cfg.Kafka.Topic,
		GroupID:  cfg.Kafka.GroupID,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	slog.Info("Connected to Kafka. Waiting for messages...", slog.String("topic", cfg.Kafka.Topic))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			//ReadMessage блокирует выполнение пока не придет новое сообщение
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				// Ошибка context.Canceled возникает при нормальном выключении сервиса, это не страшно
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("Failed to read message from Kafka", slog.String("error", err.Error()))
				continue
			}

			//Обрабатываем сообщение
			var event OrderCreatedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				slog.Error("Failed to unmarshal event", slog.String("error", err.Error()))
				continue
			}

			//Имитируем бизнес логику (отправка SMS, Email, Push)
			slog.Info("Новое уведомление!",
				slog.String("order_id", event.OrderID.String()),
				slog.String("user_id", event.UserID.String()),
				slog.Int64("total_amount", event.TotalAmount))
		}
		//В проде тут будет вызов внешнего API провайдера.
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit
	slog.Info("Shutting down worker...", slog.String("signal", sign.String()))

	// Отменяем контекст, чтобы прервать ожидание в reader.ReadMessage
	cancel()

	time.Sleep(1 * time.Second)
	slog.Info("Worker exited successfully")
}

func setupLogger(isDebug bool) {
	var logHandler slog.Handler
	if isDebug {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	slog.SetDefault(slog.New(logHandler))
}
