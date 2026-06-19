package v1

import (
	"order-service/internal/usecase"

	"github.com/go-chi/chi/v5"
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
