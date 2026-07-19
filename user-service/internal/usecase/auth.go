package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
	"user-service/internal/domain"
	"user-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Пользовательские ошибки слоя бизнес-логики
var (
	ErrInvalidEmailFormat = errors.New("invalid email format")
	ErrPasswordTooShort   = errors.New("password must be at least 6 characters long")
)

type AuthUseCase interface {
	Register(ctx context.Context, email, password string) (uuid.UUID, error)
	Login(ctx context.Context, email, password string) (string, error)
}

type authUseCase struct {
	repo      repository.UserRepository
	jwtSecret string
	jwtTTL    time.Duration
}

func NewAuthUseCase(repo repository.UserRepository, secret string, ttl time.Duration) AuthUseCase {
	return &authUseCase{
		repo: repo, jwtSecret: secret, jwtTTL: ttl,
	}
}

func (u *authUseCase) Register(ctx context.Context, email, password string) (uuid.UUID, error) {
	if email == "" {
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
		CreatedAt:    time.Now().UTC(),
	}

	if err := u.repo.Create(ctx, user); err != nil {
		return uuid.Nil, fmt.Errorf("usecase.Register create user: %w", err)
	}

	return user.ID, nil
}

func (u *authUseCase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := u.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Важное правило безопасности: не говорим "Пользователь не найден" или "Неверный пароль".
			// Отдаем общую ошибку, чтобы хакеры не могли собирать базу существующих email-ов перебором.
			return "", domain.ErrInvalidCredentials
		}
		return "", fmt.Errorf("usecase.Login get user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", domain.ErrInvalidCredentials
		}
		return "", fmt.Errorf("usecase.Login compare password: %w", err)
	}

	token, err := u.generateJWT(user)
	if err != nil {
		return "", fmt.Errorf("usecase.Login generate token: %w", err)
	}

	return token, nil
}

func (u *authUseCase) generateJWT(user *domain.User) (string, error) {
	//Создаем Payload (нагрузку) токена. В библиотеке jwt/v5 это называется Claims.
	//MapClaims позволяет положить любые JSON-поля.
	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"exp":     time.Now().Add(u.jwtTTL).Unix(), //Время протухания
		"iat":     time.Now().Unix(),               //Время выдачи
	}

	//Создаем структуру токена с алгоритмом подписи HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//Подписываем токен нашим секретным ключом (превращаем в итоговую строку)
	tokenString, err := token.SignedString([]byte(u.jwtSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
