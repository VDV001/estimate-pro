// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

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
	UpdateFlags(ctx context.Context, id string, isSigned, isFinal bool) error
	ClearFinal(ctx context.Context, documentID string) error             // clear is_final for all versions of a document
	ClearFinalByProject(ctx context.Context, projectID string) error    // clear is_final for all versions in all documents of a project
	SetTags(ctx context.Context, versionID string, tags []string) error
	GetTags(ctx context.Context, versionID string) ([]string, error)
}

type FileStorage interface {
	Upload(ctx context.Context, key string, data io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
