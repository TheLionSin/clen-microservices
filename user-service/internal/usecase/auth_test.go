package usecase

import (
	"context"
	"testing"
	"time"
	"user-service/internal/domain"
	"user-service/internal/repository/redisrepo"
	"user-service/internal/usecase/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func setupTest(t *testing.T) (*gomock.Controller, *mocks.MockUserRepository, *mocks.MockSessionRepository, AuthUseCase) {
	ctrl := gomock.NewController(t)
	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockSessionRepo := mocks.NewMockSessionRepository(ctrl)

	useCase := NewAuthUseCase(
		mockUserRepo, mockSessionRepo, "test-key",
		15*time.Minute, 24*time.Hour)

	return ctrl, mockUserRepo, mockSessionRepo, useCase
}

func TestRegister_Success(t *testing.T) {
	ctrl, mockUserRepo, _, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	email := "test@mail.ru"
	password := "asd@123#"

	// Ожидаем, что логика успешно создаст пользователя.
	// gomock.Any() используется, так как ID и CreatedAt генерируются внутри UseCase
	mockUserRepo.EXPECT().
		Create(ctx, gomock.Any()).Return(nil)

	userID, err := useCase.Register(ctx, email, password)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, userID, "ID пользователя не должен быть пустым")
}

func TestRegister_InvalidEmail(t *testing.T) {
	ctrl, _, _, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	// Моки не настраиваем, код должен отвалиться на этапе валидации регулярки

	userID, err := useCase.Register(ctx, "invalid-email", "asd@123#")

	assert.ErrorIs(t, err, ErrInvalidEmailFormat)
	assert.Equal(t, uuid.Nil, userID)
}

func TestRegister_PasswordTooShort(t *testing.T) {
	ctrl, _, _, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()

	userID, err := useCase.Register(ctx, "test@mail.ru", "12345")

	assert.ErrorIs(t, err, ErrPasswordTooShort)
	assert.Equal(t, uuid.Nil, userID)
}

func TestLogin_Success(t *testing.T) {
	ctrl, mockUserRepo, mockSessionRepo, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	email := "test@mail.ru"
	password := "asd@123#"
	userID := uuid.New()

	// Генерируем реальный хэш для мок-ответа (MinCost для скорости в тестах)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)

	expectedUser := &domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(hash),
		Role:         "user",
	}

	// 1. Ожидаем поход за юзером в Postgres
	mockUserRepo.EXPECT().GetByEmail(ctx, email).
		Return(expectedUser, nil)

	// 2. Ожидаем сохранение Refresh токена в Redis
	mockSessionRepo.EXPECT().
		SetRefreshToken(ctx, gomock.Any(), userID, 24*time.Hour).
		Return(nil)

	accessToken, refreshToken, err := useCase.Login(ctx, email, password)

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken, "Access token should be generated")
	assert.NotEmpty(t, refreshToken, "Refresh token should be generated")
}

func TestLogin_UserNotFound(t *testing.T) {
	ctrl, mockUserRepo, _, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	email := "unknown@mail.ru"

	mockUserRepo.EXPECT().GetByEmail(ctx, email).
		Return(nil, domain.ErrUserNotFound)

	accessToken, refreshToken, err := useCase.Login(ctx, email, "any-password")

	// Бизнес-логика должна скрыть реальную причину (user not found) и выдать общую ошибку
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	ctrl, mockUserRepo, _, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	email := "user@mail.ru"
	correctPassword := "asd@123#"
	wrongPassword := "wrongPassword"

	hash, err := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.MinCost)
	require.NoError(t, err)

	expectedUser := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
	}

	mockUserRepo.EXPECT().GetByEmail(ctx, email).
		Return(expectedUser, nil)

	// Сессия в Redis сохраняться не должна

	accessToken, refreshToken, err := useCase.Login(ctx, email, wrongPassword)

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
}

func TestChangePassword_Success(t *testing.T) {
	ctrl, mockUserRepo, _, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := uuid.New()
	oldPassword := "12345Pas"
	newPassword := "asd@123#"

	// Генерируем хэш старого пароля, чтобы bcrypt.CompareHashAndPassword смог его проверить
	oldHash, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.MinCost)
	require.NoError(t, err)

	expectedUser := &domain.User{
		ID:           userID,
		PasswordHash: string(oldHash),
	}

	// 1. Проверяем, существует ли юзер и достаем его старый хэш
	mockUserRepo.EXPECT().GetByID(ctx, userID).
		Return(expectedUser, nil)

	// 2. Ожидаем обновление пароля с ЛЮБЫМ новым хэшем (так как соль каждый раз новая, хэш предсказать нельзя)
	mockUserRepo.EXPECT().UpdatePassword(ctx, userID, gomock.Any()).
		Return(nil)

	err = useCase.ChangePassword(ctx, userID, oldPassword, newPassword)

	assert.NoError(t, err)
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	ctrl, mockUserRepo, _, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := uuid.New()
	oldPassword := "12345Pas"
	wrongOldPassword := "wrong123"
	newPassword := "asd@123#"

	oldHash, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.MinCost)
	require.NoError(t, err)

	expectedUser := &domain.User{
		ID:           userID,
		PasswordHash: string(oldHash),
	}

	mockUserRepo.EXPECT().GetByID(ctx, userID).
		Return(expectedUser, nil)

	// UpdatePassword не должен вызваться. Логика должна упасть на этапе проверки старого пароля.

	err = useCase.ChangePassword(ctx, userID, wrongOldPassword, newPassword)

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestGetProfile_Success(t *testing.T) {
	ctrl, mockUserRepo, _, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := uuid.New()
	expectedUser := &domain.User{
		ID:    userID,
		Email: "test@mail.ru",
	}

	mockUserRepo.EXPECT().GetByID(ctx, userID).
		Return(expectedUser, nil)

	user, err := useCase.GetProfile(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
}

func TestRefreshTokens_Success(t *testing.T) {
	ctrl, mockUserRepo, mockSessionRepo, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	refreshToken := "valid-refresh-token"
	userID := uuid.New()

	expectedUser := &domain.User{
		ID:   userID,
		Role: "user",
	}

	// 1. Ищем, чей это токен в Redis
	mockSessionRepo.EXPECT().GetUserIDByToken(ctx, refreshToken).
		Return(userID, nil)

	// 2. Удаляем старый токен (это и есть ротация для защиты от кражи)
	mockSessionRepo.EXPECT().DeleteRefreshToken(ctx, refreshToken).
		Return(nil)

	// 3. Достаем юзера из базы, чтобы забрать его актуальную роль
	mockUserRepo.EXPECT().GetByID(ctx, userID).
		Return(expectedUser, nil)

	// 4. Сохраняем НОВЫЙ токен в Redis
	mockSessionRepo.EXPECT().
		SetRefreshToken(ctx, gomock.Any(), userID, 24*time.Hour).
		Return(nil)

	newAccess, newRefresh, err := useCase.RefreshTokens(ctx, refreshToken)

	assert.NoError(t, err)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)
	assert.NotEqual(t, refreshToken, newRefresh)
}

func TestRefreshTokens_InvalidSession(t *testing.T) {
	ctrl, _, mockSessionRepo, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	invalidToken := "expired-token"

	mockSessionRepo.EXPECT().GetUserIDByToken(ctx, invalidToken).
		Return(uuid.Nil, redisrepo.ErrSessionNotFound)

	newAccess, newRefresh, err := useCase.RefreshTokens(ctx, invalidToken)

	assert.ErrorIs(t, err, ErrInvalidSession)
	assert.Empty(t, newAccess)
	assert.Empty(t, newRefresh)
}

func TestLogout_Success(t *testing.T) {
	ctrl, _, mockSessionRepo, useCase := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	refreshToken := "some-refresh-token"

	mockSessionRepo.EXPECT().
		DeleteRefreshToken(ctx, refreshToken).
		Return(nil)

	err := useCase.Logout(ctx, refreshToken)

	assert.NoError(t, err)
}
