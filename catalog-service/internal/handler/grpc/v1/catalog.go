package v1

import (
	"catalog-service/internal/domain"
	"catalog-service/internal/usecase"
	catalogv1 "catalog-service/pkg/api/catalog/v1"
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CatalogGRPCServer struct {
	catalogv1.UnimplementedCatalogServiceServer
	useCase usecase.ProductUseCase
}

func Register(gRPC *grpc.Server, u usecase.ProductUseCase) {
	catalogv1.RegisterCatalogServiceServer(gRPC, &CatalogGRPCServer{
		useCase: u,
	})
}

func (s *CatalogGRPCServer) CheckProduct(ctx context.Context, req *catalogv1.CheckProductRequest) (*catalogv1.CheckProductResponse, error) {
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id format")
	}

	product, err := s.useCase.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			// Если товара нет, мы не отдаем ошибку сервера!
			// Мы отдаем успешный ответ, но говорим exists = false
			return &catalogv1.CheckProductResponse{
				Exists: false,
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &catalogv1.CheckProductResponse{
		Exists: true,
		Price:  product.Price,
		Stock:  int32(product.Stock),
		Name:   product.Name,
	}, nil
}
