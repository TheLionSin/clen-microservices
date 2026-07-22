package usecase_test

import (
	"context"
	"errors"
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

func TestAddToCart_Success(t *testing.T) {
	//1.Инициализация контроллера моков
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	//2.Фейковые зависимости
	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)

	//3. Собираем UseCase
	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	//Тестовые данные
	ctx := context.Background()
	userID := uuid.New()
	productID := uuid.New()
	quantity := 2

	//1.Подготовка(программируем поведение наших фейков.

	// Ожидаем, что UseCase спросит у каталога наличие товара.
	// Говорим моку: верни ответ, что товар существует (Exists: true) и его 10 штук.
	mockCatalog.EXPECT().
		CheckProduct(ctx, productID.String()).
		Return(&catalogv1.CheckProductResponse{
			Exists: true,
			Stock:  10,
			Price:  5000,
		}, nil)

	// Ожидаем, что UseCase пойдет в Redis искать корзину.
	// Говорим моку: верни ошибку "Корзина не найдена" (имитируем нового пользователя).
	mockRepo.EXPECT().
		Get(ctx, userID).
		Return(nil, domain.ErrCartNotFound)

	// Ожидаем, что UseCase попытается сохранить новую корзину.
	// Говорим моку: просто верни nil (ошибок при сохранении нет).
	// gomock.Any() означает, что мы согласны на любую структуру корзины в аргументах.
	mockRepo.EXPECT().
		Save(ctx, gomock.Any()).
		Return(nil)

	//2.Действие(вызываем реальную бизнес-логику)
	input := usecase.AddToCartInput{
		UserID:    userID,
		ProductID: productID,
		Quantity:  quantity,
	}
	cart, err := useCase.AddToCart(ctx, input)

	//3.ASSERT(проверки)
	assert.NoError(t, err)
	assert.NotNil(t, cart)
	assert.Equal(t, userID, cart.UserID)
	assert.Len(t, cart.Items, 1)
	assert.Equal(t, quantity, cart.Items[0].Quantity)
}

func TestAddToCart_NotEnoughStock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)

	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	ctx := context.Background()
	userID := uuid.New()
	productID := uuid.New()

	mockCatalog.EXPECT().CheckProduct(ctx, productID.String()).
		Return(&catalogv1.CheckProductResponse{
			Exists: true,
			Stock:  1,
			Price:  5000,
		}, nil)

	// Намеренно НЕ прописываем mockRepo.EXPECT().Get() или Save()
	// Логика должна отвалиться еще на этапе проверки каталога.
	// Если код попытается пойти в Redis, тест упадет - это защищает нас от лишних запросов к БД при ошибках!
	input := usecase.AddToCartInput{
		UserID:    userID,
		ProductID: productID,
		Quantity:  5,
	}
	cart, err := useCase.AddToCart(ctx, input)

	assert.ErrorIs(t, err, usecase.ErrNotEnoughStock)
	assert.Nil(t, cart) // Корзина не должна создаться
}

func TestAddToCart_InvalidQuantity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	ctx := context.Background()

	input := usecase.AddToCartInput{
		UserID:    uuid.New(),
		ProductID: uuid.New(),
		Quantity:  0,
	}

	cart, err := useCase.AddToCart(ctx, input)

	assert.ErrorIs(t, err, usecase.ErrInvalidQuantity)
	assert.Nil(t, cart)
}

func TestAddToCart_ExistingItem_IncreasesQuantity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	ctx := context.Background()
	userID := uuid.New()
	productID := uuid.New()

	existingCart := &domain.Cart{
		UserID: userID,
		Items: []domain.CartItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	mockCatalog.EXPECT().CheckProduct(ctx, productID.String()).
		Return(&catalogv1.CheckProductResponse{
			Exists: true,
			Stock:  10,
		}, nil)

	mockRepo.EXPECT().Get(ctx, userID).Return(existingCart, nil)

	mockRepo.EXPECT().Save(ctx, gomock.Any()).Return(nil)

	input := usecase.AddToCartInput{
		UserID:    userID,
		ProductID: productID,
		Quantity:  3,
	}
	cart, err := useCase.AddToCart(ctx, input)
	assert.NoError(t, err)
	assert.NotNil(t, cart)
	assert.Equal(t, 5, cart.Items[0].Quantity)
}

func TestAddToCart_ProductNotFoundInCatalog(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	ctx := context.Background()
	productID := uuid.New()

	mockCatalog.EXPECT().CheckProduct(ctx, productID.String()).
		Return(&catalogv1.CheckProductResponse{Exists: false}, nil)

	input := usecase.AddToCartInput{
		UserID:    uuid.New(),
		ProductID: productID,
		Quantity:  1,
	}
	cart, err := useCase.AddToCart(ctx, input)

	assert.ErrorIs(t, err, usecase.ErrProductNotFound)
	assert.Nil(t, cart)
}

func TestAddToCart_CatalogError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	ctx := context.Background()
	productID := uuid.New()

	grpcErr := errors.New("grpc connection timeout")

	mockCatalog.EXPECT().CheckProduct(ctx, productID.String()).
		Return(nil, grpcErr)

	input := usecase.AddToCartInput{
		UserID:    uuid.New(),
		ProductID: productID,
		Quantity:  1,
	}
	cart, err := useCase.AddToCart(ctx, input)

	assert.Error(t, err)
	assert.ErrorIs(t, err, grpcErr)
	assert.Nil(t, cart)
}

func TestAddToCart_SaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	ctx := context.Background()
	userID := uuid.New()
	productID := uuid.New()

	mockCatalog.EXPECT().CheckProduct(ctx, productID.String()).
		Return(&catalogv1.CheckProductResponse{Exists: true, Stock: 10}, nil)

	mockRepo.EXPECT().Get(ctx, userID).
		Return(nil, domain.ErrCartNotFound)

	saveErr := errors.New("redis dick full")
	mockRepo.EXPECT().Save(ctx, gomock.Any()).Return(saveErr)

	input := usecase.AddToCartInput{
		UserID:    userID,
		ProductID: productID,
		Quantity:  1,
	}
	cart, err := useCase.AddToCart(ctx, input)

	assert.Error(t, err)
	assert.ErrorIs(t, err, saveErr)
	assert.Nil(t, cart)
}

func TestGetCart_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	ctx := context.Background()
	userID := uuid.New()

	expectedCart := &domain.Cart{
		UserID: userID,
		Items: []domain.CartItem{
			{ProductID: uuid.New(), Quantity: 2},
		},
	}

	mockRepo.EXPECT().Get(ctx, userID).
		Return(expectedCart, nil)

	cart, err := useCase.GetCart(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedCart, cart)
}

func TestGetCart_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	ctx := context.Background()
	userID := uuid.New()

	redisErr := errors.New("redis error")

	mockRepo.EXPECT().Get(ctx, userID).Return(nil, redisErr)

	cart, err := useCase.GetCart(ctx, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "usecase.GetCart")
	assert.Nil(t, cart)
}

func TestClearCart_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.EXPECT().Delete(ctx, userID).Return(nil)

	err := useCase.ClearCart(ctx, userID)

	assert.NoError(t, err)
}

func TestClearCart_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockCartRepository(ctrl)
	mockCatalog := usecasemocks.NewMockCatalogProvider(ctrl)
	useCase := usecase.NewCartUseCase(mockRepo, mockCatalog)

	ctx := context.Background()
	userID := uuid.New()

	dbErr := errors.New("redis error")

	mockRepo.EXPECT().Delete(ctx, userID).Return(dbErr)

	err := useCase.ClearCart(ctx, userID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}
