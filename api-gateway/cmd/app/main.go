package main

import (
	"api-gateway/internal/config"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	authMiddleware "api-gateway/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {

	cfg := config.GetConfig()
	setupLogger(cfg.IsDebug)
	slog.Info("Starting API Gateway", slog.String("port", cfg.Listen.Port))

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	//Создаем Reverse Proxy для каждого микросервиса
	catalogProxy := createProxy(cfg.Services.Catalog)
	orderProxy := createProxy(cfg.Services.Order)
	userProxy := createProxy(cfg.Services.User)

	// --- МАРШРУТИЗАЦИЯ ---

	//1. Публичные маршруты (Аутентификация и Каталог) - сюда пускаем всех без проверки JWT
	r.Group(func(public chi.Router) {
		//Любой запрос на /api/v1/auth/* летит в User Service
		public.Handle("/api/v1/auth/*", userProxy)

		//Запросы на /api/v1/products, /api/v1/categories летят в Catalog Service
		public.Handle("/api/v1/products*", catalogProxy)
		public.Handle("/api/v1/categories*", catalogProxy)
	})

	//2. Защищенные маршруты (Корзина и Заказы) - сюда пускаем только с валидным JWT
	r.Group(func(private chi.Router) {
		// Включаем наш "Фейсконтроль"
		private.Use(authMiddleware.Auth(cfg.JWT.Secret))

		//Все запросы корзины и чекаута летят в Order Service
		//Auth middleware уже подставит X-User-Id в заголовки)
		private.Handle("/api/v1/cart*", orderProxy)
		private.Handle("/api/v1/orders*", orderProxy)
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Listen.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Gateway server error", slog.String("error", err.Error()))
		}
	}()

	slog.Info("API Gateway is running", slog.String("addr", srv.Addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sign := <-quit
	slog.Info("Shutting down API Gateway...", slog.String("signal", sign.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Gateway forced to shutdown", slog.String("error", err.Error()))
	}
	slog.Info("API Gateway exited successfully")

}

// createProxy создает ReverseProxy, который перенаправляет запрос на указанный URL
func createProxy(targetURL string) *httputil.ReverseProxy {
	target, err := url.Parse(targetURL)
	if err != nil {
		slog.Error("Invalid target URL", slog.String("url", targetURL), slog.String("error", err.Error()))
		os.Exit(1)
	}

	return httputil.NewSingleHostReverseProxy(target)
}

func setupLogger(isDebug bool) {
	var logHandler slog.Handler
	if isDebug {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	slog.SetDefault(slog.New(logHandler))

}
