package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	router "user-service/internal/handler/http"
	"user-service/internal/repository/postgresrepo"
	"user-service/internal/usecase"
	"user-service/pkg/client/postgresql"

	"os"
	"time"
	"user-service/internal/config"
)

func main() {
	cfg := config.GetConfig()
	setupLogger(cfg.IsDebug)
	slog.Info("Starting User Service", slog.String("port", cfg.Listen.Port))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//1.Infra
	pgPool, err := postgresql.NewClient(ctx, cfg.PostgreSQL.URL)
	if err != nil {
		slog.Error("Failed to connect to Postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pgPool.Close()

	//2.Layers
	userRepo := postgresrepo.NewUserRepo(pgPool)
	authUseCase := usecase.NewAuthUseCase(userRepo, cfg.JWT.Secret, cfg.JWT.TTL)

	//3.HTTP Server
	r := router.NewRouter(authUseCase)
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
	slog.Info("User Service is running", slog.String("addr", srv.Addr))

	//4.Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit
	slog.Info("Shutting down server...", slog.String("signal", sign.String()))

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
