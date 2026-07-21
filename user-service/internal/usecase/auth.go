package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"
	"user-service/internal/domain"
	"user-service/internal/repository"
	"user-service/internal/repository/redisrepo"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Пользовательские ошибки слоя бизнес-логики
var (
	ErrInvalidEmailFormat = errors.New("invalid email format")
	ErrPasswordTooShort   = errors.New("password must be at least 6 characters long")
	ErrInvalidSession     = errors.New("invalid or expired refresh token")
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type AuthUseCase interface {
	Register(ctx context.Context, email, password string) (uuid.UUID, error)
	Login(ctx context.Context, email, password string) (string, string, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	RefreshTokens(ctx context.Context, refreshToken string) (string, string, error)
	Logout(ctx context.Context, refreshToken string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
}

type authUseCase struct {
	repo        repository.UserRepository
	sessionRepo repository.SessionRepository
	jwtSecret   string
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

func NewAuthUseCase(repo repository.UserRepository, sessionRepo repository.SessionRepository,
	secret string, accessTTL time.Duration, refreshTTL time.Duration) AuthUseCase {
	return &authUseCase{
		repo:        repo,
		sessionRepo: sessionRepo,
		jwtSecret:   secret,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
	}
}

func (u *authUseCase) Register(ctx context.Context, email, password string) (uuid.UUID, error) {
	if !emailRegex.MatchString(email) {
		return uuid.Nil, ErrInvalidEmailFormat
	}
	if len(password) < 6 {
		return uuid.Nil, ErrPasswordTooShort
	}

	// bcrypt.DefaultCost = 10. Это оптимальный баланс между скоростью и безопасностью.
	// Чем выше cost, тем дольше генерируется хэш, что усложняет брутфорс (подбор паролей хакерами).
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, fmt.Errorf("usecase.Register hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         "user",
		CreatedAt:    time.Now().UTC(),
	}

	if err := u.repo.Create(ctx, user); err != nil {
		return uuid.Nil, fmt.Errorf("usecase.Register create user: %w", err)
	}

	return user.ID, nil
}

func (u *authUseCase) Login(ctx context.Context, email, password string) (string, string, error) {
	if !emailRegex.MatchString(email) {
		return "", "", domain.ErrInvalidCredentials
	}
	user, err := u.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Важное правило безопасности: не говорим "Пользователь не найден" или "Неверный пароль".
			// Отдаем общую ошибку, чтобы хакеры не могли собирать базу существующих email-ов перебором.
			return "", "", domain.ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("usecase.Login get user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", "", domain.ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("usecase.Login compare password: %w", err)
	}

	// Генерируем ПАРУ токенов
	accessToken, refreshToken, err := u.generateTokens(ctx, user.ID, user.Role)
	if err != nil {
		return "", "", fmt.Errorf("usecase.Login generate tokens: %w", err)
	}

	return accessToken, refreshToken, nil

}

func (u *authUseCase) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := u.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase.GetProfile: %w", err)
	}
	return user, nil
}

func (u *authUseCase) RefreshTokens(ctx context.Context, refreshToken string) (string, string, error) {
	//1. Ищем Refresh Token в Redis. Если его там нет (логаут или истек TTL) - ошибка.
	userID, err := u.sessionRepo.GetUserIDByToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, redisrepo.ErrSessionNotFound) {
			return "", "", ErrInvalidSession
		}
		return "", "", fmt.Errorf("usecase.RefreshTokens get session: %w", err)
	}

	// 2. Для максимальной безопасности (Rotated Refresh Tokens),
	// удаляем старый рефреш токен и выдаем полностью новую пару.
	// Это защищает от кражи рефреш-токена.
	_ = u.sessionRepo.DeleteRefreshToken(ctx, refreshToken)

	user, err := u.repo.GetByID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("usecase.RefreshTokens get user: %w", err)
	}

	// 3. Генерируем новые токены
	newAccess, newRefresh, err := u.generateTokens(ctx, userID, user.Role)
	if err != nil {
		return "", "", fmt.Errorf("usecase.RefreshTokens generate: %w", err)
	}

	return newAccess, newRefresh, nil
}

func (u *authUseCase) Logout(ctx context.Context, refreshToken string) error {
	err := u.sessionRepo.DeleteRefreshToken(ctx, refreshToken)
	if err != nil {
		return fmt.Errorf("usecase.Logout: %w", err)
	}
	return nil
}

func (u *authUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return ErrPasswordTooShort
	}

	user, err := u.repo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("usecase.ChangePassword get user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return domain.ErrInvalidCredentials
		}
		return fmt.Errorf("usecase.ChangePassword compare old : %w", err)
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("usecase.ChangePassword hash new: %w", err)
	}

	err = u.repo.UpdatePassword(ctx, userID, string(newHash))
	if err != nil {
		return fmt.Errorf("usecase.ChangePassword update: %w", err)
	}
	return nil
}

func (u *authUseCase) generateTokens(ctx context.Context, userID uuid.UUID, role string) (string, string, error) {
	//Создаем Payload (нагрузку) токена. В библиотеке jwt/v5 это называется Claims.
	//MapClaims позволяет положить любые JSON-поля.
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"role":    role,
		"exp":     time.Now().Add(u.accessTTL).Unix(), //Время протухания
		"iat":     time.Now().Unix(),                  //Время выдачи
	}

	//Создаем структуру токена с алгоритмом подписи HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	//Подписываем access токен нашим секретным ключом (превращаем в итоговую строку)
	accessToken, err := token.SignedString([]byte(u.jwtSecret))
	if err != nil {
		return "", "", err
	}

	refreshToken := uuid.New().String()

	err = u.sessionRepo.SetRefreshToken(ctx, refreshToken, userID, u.refreshTTL)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil

}
