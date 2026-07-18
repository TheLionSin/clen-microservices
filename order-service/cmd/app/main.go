package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"order-service/internal/config"
	router "order-service/internal/handler/http"
	"order-service/internal/repository/postgresrepo"
	"order-service/internal/repository/redisrepo"
	"order-service/internal/usecase"
	grpcclient "order-service/pkg/client/grpc"
	"order-service/pkg/client/kafka"
	"order-service/pkg/client/postgresql"
	"order-service/pkg/client/redis"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	cfg := config.GetConfig()
	setupLogger(cfg.IsDebug)
	slog.Info("Starting Order Service", slog.String("port", cfg.Listen.Port))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// --- 1. Инфраструктура ---
	// Подключаемся к Redis
	redisClient, err := redis.NewClient(ctx, cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.DB)

	if err != nil {
		slog.Error("Failed to connect to Redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer redisClient.Close()

	// Подключаемся к Catalog Service по gRPC
	catalogGRPC, err := grpcclient.NewCatalogClient(cfg.Clients.CatalogGRPC)
	if err != nil {
		slog.Error("Failed to connect to Catalog gRPC", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Подключение к PostgreSQL
	pgPool, err := postgresql.NewClient(ctx, cfg.PostgreSQL.URL)
	if err != nil {
		slog.Error("Failed to connect to Postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pgPool.Close()

	// Подключение к Kafka
	// В конфиге Brokers это массив, но из ENV читается строка.
	// Для простоты MVP передаем одним куском, cleanenv сам разобьет по запятым, если их несколько.
	kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	defer kafkaProducer.Close()

	// --- 2. Слои приложения ---
	cartRepo := redisrepo.NewCartRepo(redisClient)
	orderRepo := postgresrepo.NewOrderRepo(pgPool)
	cartUseCase := usecase.NewCartUseCase(cartRepo, catalogGRPC)
	orderUseCase := usecase.NewOrderUseCase(cartRepo, orderRepo, catalogGRPC, kafkaProducer)

	// --- 3. HTTP Сервер ---
	r := router.NewRouter(cartUseCase, orderUseCase)
	srv := &http.Server{
		Addr:         ":" + cfg.Listen.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", slog.String("error", err.Error()))
		}
	}()
	slog.Info("Order Service is running", slog.String("addr", srv.Addr))

	// --- 4. Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit
	slog.Info("Shutting down server gracefully...", slog.String("signal", sign.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", slog.String("error", err.Error()))

	}

	slog.Info("Server exited successfully")
}

func setupLogger(isDebug bool) {
	var logHandler slog.Handler
	if isDebug {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)
}
