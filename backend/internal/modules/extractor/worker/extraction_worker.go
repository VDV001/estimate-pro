// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package worker hosts the asynchronous body of the extractor
// pipeline. The Process method orchestrates: load extraction by
// ID, transition to processing, fetch document bytes from the
// document storage adapter, dispatch through shared/reader to
// extract plain text, run a prompt-injection guard, call the
// shared/llm completer with a JSON-mode prompt, validate the
// response schema, persist the extracted tasks, and stamp the
// completed audit event. PR-B3 builds the body slice by slice
// under TDD; this file grows commit by commit.
package worker

import (
	"context"
	"fmt"
)

// ExtractionArgs is the JSON payload enqueued onto the river queue
// for a single extraction job. The shape is pinned across the
// PR-B2 / PR-B3 boundary so the queue contract stays stable.
type ExtractionArgs struct {
	ExtractionID string `json:"extraction_id"`
}

// ExtractionWorker is the river-compatible worker for processing
// extraction jobs. Dependencies are injected at construction so the
// worker can be unit-tested with panic-on-unexpected-call fakes
// and integration-tested against postgres + minio + httptest LLM.
type ExtractionWorker struct {
	store    ExtractionStore
	source   DocumentSource
	reader   TextExtractor
	llm      Completer
	security SecurityChecker
}

// NewExtractionWorker wires the worker with its five collaborators.
// All five are required — passing nil for any of them is a
// programmer error and will surface as a nil-pointer panic the
// first time the corresponding code path executes. The composition
// root in cmd/server/main.go is the single instantiation site,
// gated behind FEATURE_DOCUMENT_PIPELINE_ENABLED.
func NewExtractionWorker(store ExtractionStore, source DocumentSource, reader TextExtractor, llm Completer, security SecurityChecker) *ExtractionWorker {
	return &ExtractionWorker{
		store:    store,
		source:   source,
		reader:   reader,
		llm:      llm,
		security: security,
	}
}

// Process is invoked by the river job runner for each enqueued
// ExtractionArgs. PR-B3 builds the body slice by slice — currently
// the load-and-return shell. Subsequent commits add the
// transition / download / parse / security / LLM / persist stages
// each behind their own RED+GREEN pair.
func (w *ExtractionWorker) Process(ctx context.Context, args ExtractionArgs) error {
	if _, err := w.store.GetByID(ctx, args.ExtractionID); err != nil {
		return fmt.Errorf("worker.Process load extraction %q: %w", args.ExtractionID, err)
	}
	return nil
}
