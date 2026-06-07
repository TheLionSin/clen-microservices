package http

import (
	v1 "catalog-service/internal/handler/http/v1"
	"catalog-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter Собирает весь HTTP API
func NewRouter(productUseCase usecase.ProductUseCase) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	productHandler := v1.NewProductHandler(productUseCase)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/products", func(r chi.Router) {
			productHandler.Register(r)
		})
	})
	
	return r
}
