package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const testSecretKey = "asd@123#"

// generateTestToken - приватный хелпер для создания валидных и протухших JWT токенов
func generateTestToken(t *testing.T, userID, role string, expDuration time.Duration) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(expDuration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecretKey))
	assert.NoError(t, err)
	return tokenString
}

// dummyHandler имитирует следующий хендлер в цепочке (например, переход к бизнес-логике).
// Если запрос дошел сюда, значит Middleware его пропустил.
var dummyHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestAuthMiddleware_Success(t *testing.T) {
	//Валидный токен
	userID := "123e4567-e89b-12d3-a456-426614174000"
	role := "user"
	validToken := generateTestToken(t, userID, role, 15*time.Minute)

	//Виртуальный HTTP-запрос
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)

	// Рекордер запишет всё, что ответит Middleware (статус, заголовки, тело)
	recorder := httptest.NewRecorder()

	//Оборачиваем пустой dummyHandler в Auth Middleware
	handler := Auth(testSecretKey)(dummyHandler)

	//Запускаем запрос
	handler.ServeHTTP(recorder, req)

	//Результаты
	assert.Equal(t, http.StatusOK, recorder.Code)

	// Middleware должен был вырезать заголовок Authorization и подставить внутренние заголовки
	assert.Empty(t, req.Header.Get("Authorization"), "Заголовок Authorization должен быть удален для безопасности")
	assert.Equal(t, userID, req.Header.Get("X-User-Id"), "Middleware должен был прокинуть X-User-Id")
	assert.Equal(t, role, req.Header.Get("X-User-Role"))
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "JustTokenWithoutBearer")
	recorder := httptest.NewRecorder()

	handler := Auth(testSecretKey)(dummyHandler)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code, "Ожидаем 401 при неверном формате")
	assert.Contains(t, recorder.Body.String(), "invalid authorization header format")
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	//Протух 1 час назад
	expiredToken := generateTestToken(t, "user123", "user", -1*time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	recorder := httptest.NewRecorder()

	handler := Auth(testSecretKey)(dummyHandler)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "invalid or expired token")

}

func TestRequireAdmin_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/panel", nil)
	// Имитируем, что предыдущий Auth Middleware уже отработал и повесил заголовок
	req.Header.Set("X-User-Role", "admin")

	recorder := httptest.NewRecorder()
	handler := RequireAdmin(dummyHandler)

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestRequireAdmin_Forbidden_UserRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/panel", nil)

	req.Header.Set("X-User-Role", "user")

	recorder := httptest.NewRecorder()
	handler := RequireAdmin(dummyHandler)

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "admin role required")
}

func TestRequireAdmin_Forbidden_NoRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/panel", nil)

	recorder := httptest.NewRecorder()
	handler := RequireAdmin(dummyHandler)

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
}
