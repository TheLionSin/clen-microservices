package v1

import (
	"errors"
	"net/http"
	"order-service/internal/domain"
	"order-service/internal/usecase"

	"github.com/go-chi/chi/v5"
)

type OrderHandler struct {
	useCase usecase.OrderUseCase
}

func NewOrderHandler(u usecase.OrderUseCase) *OrderHandler {
	return &OrderHandler{useCase: u}
}

func (h *OrderHandler) Register(r chi.Router) {
	r.Post("/checkout", h.Checkout)
}

func (h *OrderHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	link, err := h.useCase.Checkout(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrCartNotFound) || errors.Is(err, usecase.ErrEmptyCart) {
			writeJSONError(w, http.StatusBadRequest, "cart is empty")
			return
		}

		// Для MVP отдаем текст ошибки клиенту (чтобы видеть, если товара не хватило)
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"whatsapp_link": link,
	})
}
