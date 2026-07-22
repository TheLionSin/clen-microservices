package usecase_test

import (
	"context"
	"order-service/internal/domain"
	"order-service/internal/repository/mocks"
	"order-service/internal/usecase"
	usecasemocks "order-service/internal/usecase/mocks"
	catalogv1 "order-service/pkg/api/catalog/v1"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCheckout_Success(t *testing.T) {
	//Инициализация контроллера и все 4 мока
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartRepo := mocks.NewMockCartRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	mockProducer := usecasemocks.NewMockMessageProducer(ctrl)

	//Собираем OrderUseCase
	useCase := usecase.NewOrderUseCase(mockCartRepo, mockOrderRepo, mockCatalog, mockProducer)

	ctx := context.Background()
	userID := uuid.New()
	productID := uuid.New()

	//Тестовая корзина
	cart := &domain.Cart{
		UserID: userID,
		Items: []domain.CartItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	//1. Сначала UseCase пойдет за корзиной
	mockCartRepo.EXPECT().Get(ctx, userID).Return(cart, nil)

	//2. Затем в каталог проверять наличие
	mockCatalog.EXPECT().CheckProduct(ctx, productID.String()).
		Return(&catalogv1.CheckProductResponse{
			Exists: true,
			Stock:  10,
			Price:  1500,
			Name:   "Test product",
		}, nil)

	//3. Попытается сохранить заказ в PostgreSQL
	mockOrderRepo.EXPECT().CreateOrder(ctx, gomock.Any()).Return(nil)
	//4. Очистит корзину в Redis
	mockCartRepo.EXPECT().Delete(ctx, userID).Return(nil)
	//5. Отправит сообщение в Kafka
	mockProducer.EXPECT().PublishOrderCreated(ctx, gomock.Any()).Return(nil)

	//Запускаем оформление
	waLink, err := useCase.Checkout(ctx, userID)

	assert.NoError(t, err)
	assert.NotEmpty(t, waLink)
	assert.Contains(t, waLink, "wa.me")
	assert.Contains(t, waLink, "3000")
}

func TestCheckout_EmptyCart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartRepo := mocks.NewMockCartRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	mockProducer := usecasemocks.NewMockMessageProducer(ctrl)

	useCase := usecase.NewOrderUseCase(mockCartRepo, mockOrderRepo, mockCatalog, mockProducer)

	ctx := context.Background()
	userID := uuid.New()

	emptyCart := &domain.Cart{
		UserID: userID,
		Items:  []domain.CartItem{},
	}

	mockCartRepo.EXPECT().Get(ctx, userID).Return(emptyCart, nil)

	waLink, err := useCase.Checkout(ctx, userID)

	assert.ErrorIs(t, err, usecase.ErrEmptyCart)
	assert.Empty(t, waLink)

}
