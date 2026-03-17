package repository

import (
	"context"
	"fmt"
	"io"

	"github.com/daniilrusanov/estimate-pro/backend/internal/infra/s3"
)

type S3FileStorage struct {
	client *s3.Client
}

func NewS3FileStorage(client *s3.Client) *S3FileStorage {
	return &S3FileStorage{client: client}
}

func (s *S3FileStorage) Upload(ctx context.Context, key string, data io.Reader, size int64, contentType string) error {
	if err := s.client.Upload(ctx, key, data, size, contentType); err != nil {
		return fmt.Errorf("fileStorage.Upload: %w", err)
	}
	return nil
}

func (s *S3FileStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	reader, err := s.client.Download(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("fileStorage.Download: %w", err)
	}
	return reader, nil
}

func (s *S3FileStorage) Delete(ctx context.Context, key string) error {
	if err := s.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("fileStorage.Delete: %w", err)
	}
	return nil
}
