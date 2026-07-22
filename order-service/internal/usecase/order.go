package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"order-service/internal/domain"
	"order-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyCart = errors.New("cannot checkout an empty card")
)

type MessageProducer interface {
	PublishOrderCreated(ctx context.Context, event any) error
}

type OrderUseCase interface {
	Checkout(ctx context.Context, userID uuid.UUID) (string, error)
	GetMyOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
}

type orderUseCase struct {
	cartRepo      repository.CartRepository
	orderRepo     repository.OrderRepository
	catalogClient CatalogProvider
	producer      MessageProducer
}

func NewOrderUseCase(cartRepo repository.CartRepository, orderRepo repository.OrderRepository,
	catalogClient CatalogProvider, producer MessageProducer) OrderUseCase {
	return &orderUseCase{
		cartRepo:      cartRepo,
		orderRepo:     orderRepo,
		catalogClient: catalogClient,
		producer:      producer,
	}
}

func (u *orderUseCase) Checkout(ctx context.Context, userID uuid.UUID) (string, error) {
	//1. Достаем корзину из Redis
	cart, err := u.cartRepo.Get(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("usecase.Checkout get cart: %w", err)
	}

	if len(cart.Items) == 0 {
		return "", ErrEmptyCart
	}

	//2. Инициализируем базовый заказ
	order := &domain.Order{
		ID:        uuid.New(),
		UserID:    userID,
		Status:    domain.StatusCreated,
		Items:     make([]domain.OrderItem, 0, len(cart.Items)),
		CreatedAt: time.Now().UTC(),
	}

	// Подготавливаем текст для WhatsApp (чтобы менеджеру было удобно читать)
	waText := fmt.Sprintf("Здравствуйте! Хочу оформить заказ №%s\n\n", order.ID.String()[:8])
	var totalAmount int64 = 0

	//3. Проходим по каждому товару из корзины и проверяем актуальность в Каталоге
	for _, cartItem := range cart.Items {
		resp, err := u.catalogClient.CheckProduct(ctx, cartItem.ProductID.String())
		if err != nil {
			return "", fmt.Errorf("usecase.Checkout check catalog: %w", err)
		}

		// Защита от того, что товар удалили или раскупили, пока он лежал в корзине
		if !resp.Exists {
			return "", fmt.Errorf("товар %s больше не доступен", cartItem.ProductID)
		}
		if resp.Stock < int32(cartItem.Quantity) {
			return "", fmt.Errorf("недостаточно товара '%s' на складе (осталось %d)", resp.Name, resp.Stock)
		}

		orderItem := domain.OrderItem{
			ID:        uuid.New(),
			OrderID:   order.ID,
			ProductID: cartItem.ProductID,
			Quantity:  cartItem.Quantity,
			Price:     resp.Price, // Берем свежую цену из каталога
		}
		order.Items = append(order.Items, orderItem)

		// Считаем сумму
		itemTotal := resp.Price * int64(cartItem.Quantity)
		totalAmount += itemTotal

		// Добавляем строчку в сообщение
		waText += fmt.Sprintf("- %s (x%d) = %d тг.\n", resp.Name, cartItem.Quantity, itemTotal)
	}

	order.TotalAmount = totalAmount
	waText += fmt.Sprintf("\nИтого к оплате: %d тг.", totalAmount)

	//4. Сохраняем заказ в PostgreSQL (сработает наша SQL-Транзакция)
	if err := u.orderRepo.CreateOrder(ctx, order); err != nil {
		return "", fmt.Errorf("usecase.Checkout create order: %w", err)
	}

	//5. Очищаем корзину в Redis, так как заказ успешно оформлен
	if err := u.cartRepo.Delete(ctx, userID); err != nil {
		// Ошибку удаления корзины мы просто логируем. Заказ уже создан,
		// не стоит отменять его только из-за сбоя очистки кэша.
		slog.Warn("Failed to clear cart after checkout", slog.String("user_id", userID.String()), slog.String("error", err.Error()))
	}

	// Отправляем событие в Kafka

	eventItems := make([]domain.OrderItemEvent, 0, len(order.Items))
	for _, item := range order.Items {
		eventItems = append(eventItems, domain.OrderItemEvent{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	event := domain.OrderCreatedEvent{
		OrderID:     order.ID,
		UserID:      order.UserID,
		TotalAmount: order.TotalAmount,
		Items:       eventItems,
		CreatedAt:   order.CreatedAt,
	}

	// Отправка в брокер сообщений не должна влиять на ответ юзеру.
	// Даже если Kafka временно упадет, заказ в базе уже есть.
	// Поэтому ошибку мы только логируем, но клиенту отдаем успешную ссылку.

	if err := u.producer.PublishOrderCreated(ctx, event); err != nil {
		slog.Error("Failed to publish order event to Kafka",
			slog.String("order_id", order.ID.String()), slog.String("error", err.Error()),
		)
	} else {
		slog.Info("Order event successfully published to Kafka", slog.String("order_id", order.ID.String()))

	}

	//6. Формируем безопасную ссылку WhatsApp.
	//url.QueryEscape заменяет пробелы на %20 и переносы строк на %0A
	WaLink := fmt.Sprintf("https://wa.me/77076665544?text=%s", url.QueryEscape(waText))

	return WaLink, nil
}

func (u *orderUseCase) GetMyOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	orders, err := u.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase.GetMyOrders: %w", err)
	}
	return orders, nil
}
