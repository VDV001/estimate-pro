// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package worker

import (
	"context"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

// ExtractionStore is the persistence port consumed by the worker.
// It is a strict subset of usecase.ExtractionRepository — only the
// methods the worker actually invokes during a single job. The
// PostgresExtractionRepository in the repository package satisfies
// it structurally; no shared abstraction is exported from domain.
type ExtractionStore interface {
	GetByID(ctx context.Context, id string) (*domain.Extraction, error)
	UpdateStatus(ctx context.Context, ext *domain.Extraction, ev *domain.ExtractionEvent) error
	SaveTasks(ctx context.Context, extractionID string, tasks []domain.ExtractedTask) error
}

// DocumentSource resolves a DocumentVersionID to its raw bytes plus
// a filename hint used by the reader composite for format dispatch.
// The extractor module does not know storage keys, file types, or
// MinIO buckets — the adapter in cmd/server/main.go composes the
// document repository (FileKey / FileType lookup) with the S3
// storage client to satisfy this port. Cross-module coupling is
// quarantined to the composition root.
type DocumentSource interface {
	Fetch(ctx context.Context, documentVersionID string) (data []byte, filename string, err error)
}

// TextExtractor lifts plain text out of in-memory document bytes.
// The shared/reader.Composite implementation satisfies it via
// extension-driven dispatch over PDF / DOCX / MD / TXT / CSV /
// XLSX readers, plus the ErrFileTooLarge size guard. The worker
// owns this port (rather than importing reader.DocumentReader
// directly) so the test fakes can panic on unexpected calls.
type TextExtractor interface {
	Parse(ctx context.Context, filename string, data []byte) (string, error)
}

// Completer is re-exported from shared/llm so tests can mock the
// LLM call without dragging the adapter zoo (Claude / OpenAI /
// Grok / Ollama) into worker_test.
type Completer = sharedllm.Completer

// SecurityChecker hides shared/security behind a worker-owned port
// for the same reason as TextExtractor — strict mock control in
// unit tests, single dependency direction in main.go.
type SecurityChecker interface {
	IsPromptInjection(text string) bool
}
