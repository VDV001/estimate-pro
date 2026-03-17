package domain

import (
	"context"
	"io"
)

type DocumentRepository interface {
	Create(ctx context.Context, doc *Document) error
	GetByID(ctx context.Context, id string) (*Document, error)
	ListByProject(ctx context.Context, projectID string) ([]*Document, error)
	Delete(ctx context.Context, id string) error
}

type VersionRepository interface {
	Create(ctx context.Context, version *DocumentVersion) error
	GetByID(ctx context.Context, id string) (*DocumentVersion, error)
	ListByDocument(ctx context.Context, documentID string) ([]*DocumentVersion, error)
	GetLatestByDocument(ctx context.Context, documentID string) (*DocumentVersion, error)
}

type FileStorage interface {
	Upload(ctx context.Context, key string, data io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
