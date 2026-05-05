// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package usecase orchestrates the extractor module's business
// flows: request, fetch, cancel, retry. Port interfaces (this file)
// live alongside the use-cases that consume them so the dependency
// rule points away from the domain — repository.PostgresExtractionRepository
// implements ExtractionRepository structurally, no shared abstraction
// is exported from domain/.
package usecase

import (
	"context"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
)

// JobEnqueuer is the async-dispatch port consumed by RequestExtraction
// and RetryExtraction. The composition root in cmd/server/main.go
// provides a river-backed implementation; test code passes a fake or
// nil (nil = skip silently, acceptable in unit tests for unrelated
// paths).
type JobEnqueuer interface {
	Enqueue(ctx context.Context, extractionID string) error
}

// ExtractionRepository is the persistence port consumed by the
// Extractor use-cases. The postgres implementation in
// repository/postgres.go satisfies it structurally.
type ExtractionRepository interface {
	Create(ctx context.Context, ext *domain.Extraction) error
	GetByID(ctx context.Context, id string) (*domain.Extraction, error)
	GetActiveByDocumentVersion(ctx context.Context, documentID, documentVersionID string) (*domain.Extraction, error)
	UpdateStatus(ctx context.Context, ext *domain.Extraction, ev *domain.ExtractionEvent) error
	SaveTasks(ctx context.Context, extractionID string, tasks []domain.ExtractedTask) error
	GetEvents(ctx context.Context, extractionID string) ([]*domain.ExtractionEvent, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Extraction, error)
}
