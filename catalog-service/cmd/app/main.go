package main

import (
	"catalog-service/internal/config"
	grpcv1 "catalog-service/internal/handler/grpc/v1"
	router "catalog-service/internal/handler/http"
	"catalog-service/internal/repository/postgres"
	"catalog-service/internal/usecase"
	"catalog-service/pkg/client/minio"
	"catalog-service/pkg/client/postgresql"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

func main() {
	os.Setenv("PG_URL", "postgres://clen_user:clenshop@localhost:5433/clen_catalog?sslmode=disable")

	cfg := config.GetConfig()
	setupLogger(cfg.IsDebug)

	slog.Info("Starting Catalog Service",
		slog.String("port", cfg.Listen.HTTPPort),
		slog.String("env", "debug"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pgPool, err := postgresql.NewClient(ctx, cfg.PostgreSQL.URL)
	if err != nil {
		slog.Error("Не удалось подключиться к БД", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer pgPool.Close()

	minioClient, err := minio.NewClient(ctx, cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, cfg.MinIO.BucketName,
		cfg.MinIO.UseSSL)

	if err != nil {
		slog.Error("Failed to connect to MinIO", slog.String("error", err.Error()))
		os.Exit(1)
	}

	//Repo
	productRepo := postgres.NewProductRepo(pgPool)
	categoryRepo := postgres.NewCategoryRepo(pgPool)
	slog.Info("Repository layer initialized successfully")
	//UseCase
	//Product
	productUseCase := usecase.NewProduct(productRepo)
	//Category
	categoryUseCase := usecase.NewCategoryUseCase(categoryRepo)
	slog.Info("UseCase layer initialized successfully")
	//MinIO
	imageUseCase := usecase.NewImageUseCase(minioClient, cfg.MinIO.BucketName, cfg.MinIO.Endpoint)
	//Router
	r := router.NewRouter(productUseCase, imageUseCase, categoryUseCase)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Listen.HTTPPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("HTTP server is running", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", slog.String("error", err.Error()))
		}
	}()

	grpcServer := grpc.NewServer()
	grpcv1.Register(grpcServer, productUseCase)

	l, err := net.Listen("tcp", ":"+cfg.Listen.GRPCPort)
	if err != nil {
		slog.Error("Failed to listen for gRPC", slog.String("error", err.Error()))
		os.Exit(1)
	}

	go func() {
		slog.Info("gRPC server running", slog.String("port", cfg.Listen.GRPCPort))
		if err := grpcServer.Serve(l); err != nil {
			slog.Error("gRPC Server error", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit
	slog.Info("Shutting down servers gracefully", slog.String("signal", sign.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP Server forced to shutdown", slog.String("error", err.Error()))
	}

	grpcServer.GracefulStop()

	slog.Info("Servers exited successfully")

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
