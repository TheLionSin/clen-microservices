package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"order-service/internal/domain"
	"order-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CartHandler struct {
	useCase usecase.CartUseCase
}

func NewCartHandler(useCase usecase.CartUseCase) *CartHandler {
	return &CartHandler{useCase: useCase}
}

func (h *CartHandler) Register(r chi.Router) {
	// Все эти роуты будут защищены (в будущем) на уровне API Gateway.
	// Пока мы доверяем заголовку X-User-Id.
	r.Post("/add", h.AddToCart)
	r.Get("/", h.GetCart)
	r.Delete("/", h.ClearCart)
}

type AddToCartRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// getUserID - внутренний хелпер для извлечения ID из заголовка
func getUserID(r *http.Request) (uuid.UUID, error) {
	userIDStr := r.Header.Get("X-User-Id")
	if userIDStr == "" {
		return uuid.Nil, errors.New("missing X-User-Id header")
	}
	return uuid.Parse(userIDStr)
}

func (h *CartHandler) AddToCart(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: invalid or missing user id")
		return
	}

	var req AddToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json format")
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid product_id format")
		return
	}

	input := usecase.AddToCartInput{
		UserID:    userID,
		ProductID: productID,
		Quantity:  req.Quantity,
	}

	cart, err := h.useCase.AddToCart(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidQuantity):
			writeJSONError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, usecase.ErrProductNotFound):
			writeJSONError(w, http.StatusNotFound, "product not found in catalog")
		case errors.Is(err, usecase.ErrNotEnoughStock):
			writeJSONError(w, http.StatusConflict, "not enough stock")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal server error")

		}
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cart, err := h.useCase.GetCart(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrCartNotFound) {
			writeJSON(w, http.StatusOK, domain.Cart{
				UserID: userID, Items: []domain.CartItem{},
			})
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, cart)
}

func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.useCase.ClearCart(r.Context(), userID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Хелперы для JSON ---
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
