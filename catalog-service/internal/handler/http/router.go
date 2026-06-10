package http

import (
	v1 "catalog-service/internal/handler/http/v1"
	"catalog-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter Собирает весь HTTP API
func NewRouter(productUseCase usecase.ProductUseCase, imageUseCase usecase.ImageUseCase,
	categoryUseCase usecase.CategoryUseCase) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	productHandler := v1.NewProductHandler(productUseCase)
	imageHandler := v1.NewImageHandler(imageUseCase)
	categoryHandler := v1.NewCategoryHandler(categoryUseCase)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/products", func(r chi.Router) {
			productHandler.Register(r)
		})
		r.Route("/images", func(r chi.Router) {
			imageHandler.Register(r)
		})
		r.Route("/category", func(r chi.Router) {
			categoryHandler.Register(r)
		})
	})

	return r
}
