package minio

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewClient(ctx context.Context, endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*minio.Client, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации MinIO клиента : %w", err)
	}

	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки бакета: %w", err)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("ошибка создания бакета: %w", err)
		}
		slog.Info("Создан новый MinIO бакет", slog.String("bucket", bucketName))

		// В идеале здесь еще нужно задать Bucket Policy на чтение (Public Read),
		// чтобы фронтенд мог просто по URL скачивать картинки.
		// Для MVP пока опустим политику, настроим через UI MinIO при необходимости.
	}

	slog.Info("Успешное подключение к MinIO")
	return minioClient, nil
}
