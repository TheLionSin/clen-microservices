package grpcclient

import (
	"context"
	"fmt"
	"log/slog"
	catalogv1 "order-service/pkg/api/catalog/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type CatalogClient struct {
	api catalogv1.CatalogServiceClient
}

func NewCatalogClient(addr string) (*CatalogClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания gRPC клиента: %w", err)
	}

	slog.Info("Успешное подключение к gRPC каталога", slog.String("addr", addr))

	return &CatalogClient{
		api: catalogv1.NewCatalogServiceClient(conn),
	}, nil
}

func (c *CatalogClient) CheckProduct(ctx context.Context, productID string) (*catalogv1.CheckProductResponse, error) {
	req := &catalogv1.CheckProductRequest{
		ProductId: productID,
	}

	resp, err := c.api.CheckProduct(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc CheckProduct: %w", err)
	}

	return resp, nil
}
