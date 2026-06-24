package http

import (
	v1 "order-service/internal/handler/http/v1"
	"order-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(cartUseCase usecase.CartUseCase) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	cartHandler := v1.NewCartHandler(cartUseCase)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/cart", func(r chi.Router) {
			cartHandler.Register(r)
		})
	})

	return r
}
