package http

import (
	v1 "user-service/internal/handler/http/v1"
	"user-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(authUseCase usecase.AuthUseCase) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	authHandler := v1.NewAuthHandler(authUseCase)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})
		r.Route("/users", func(r chi.Router) {
			r.Get("/me", authHandler.GetProfile)
			r.Put("/password", authHandler.ChangePassword)
		})
	})

	return r
}
