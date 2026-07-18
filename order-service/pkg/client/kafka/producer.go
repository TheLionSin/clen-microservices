package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer - обертка над писателем (Writer) Kafka
type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{}, // Алгоритм распределения нагрузки между партициями
		// В проде обязательно настраивают ретраи и асинхронную запись,
		// segmentio/kafka-go делает это под капотом по умолчанию
		AllowAutoTopicCreation: true,
		MaxAttempts:            3,
		ReadTimeout:            5 * time.Second,
	}

	slog.Info("Инициализирован Kafka Producer", slog.String("topic", topic))
	return &Producer{writer: w}
}

// Close закрывает соединение (нужно для Graceful Shutdown)
func (p *Producer) Close() error {
	return p.writer.Close()
}

// PublishOrderCreated конвертирует событие в JSON и шлет в Kafka
func (p *Producer) PublishOrderCreated(ctx context.Context, event any) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга события: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte("order_event"), // Ключ нужен для сохранения порядка сообщений
		Value: bytes,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("ошибка отправки сообщения в Kafka: %w", err)
	}

	return nil
}
