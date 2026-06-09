package usecase

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

var (
	ErrInvalidFileType = fmt.Errorf("разрешены только форматы JPEG и PNG")
)

type ImageUseCase interface {
	Upload(ctx context.Context, file io.Reader, fileSize int64, originalFileName string) (string, error)
}

type imageUseCase struct {
	minioClient *minio.Client
	bucketName  string
	endpoint    string
}

func NewImageUseCase(client *minio.Client, bucket, endpoint string) ImageUseCase {
	return &imageUseCase{
		minioClient: client,
		bucketName:  bucket,
		endpoint:    endpoint,
	}
}

func (u *imageUseCase) Upload(ctx context.Context, file io.Reader, fileSize int64, originalFileName string) (string, error) {
	ext := strings.ToLower(filepath.Ext(originalFileName))

	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return "", ErrInvalidFileType
	}

	newFilename := uuid.New().String() + ext

	contentType := "image/jpeg"
	if ext == ".png" {
		contentType = "image/png"
	}

	_, err := u.minioClient.PutObject(ctx, u.bucketName, newFilename, file, fileSize,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("ошибка загрузки файла в MinIO: %w", err)
	}

	// Для MVP (локалки) формируем URL вручную. В проде это будет домен CDN.
	fileURL := fmt.Sprintf("http://%s/%s/%s", u.endpoint, u.bucketName, newFilename)

	return fileURL, nil
}
