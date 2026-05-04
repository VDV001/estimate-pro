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
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
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
// the load + status-guard + pending->processing transition.
// Subsequent commits add the download / parse / security / LLM /
// persist stages each behind their own RED+GREEN pair.
//
// Idempotency: if the extraction has already moved past pending
// (processing / completed / failed / cancelled), Process returns
// nil without side effects. River may re-dispatch the same args
// after a worker crash, and the second invocation must observe
// the current state and exit cleanly.
//
// actor is hard-coded to "worker" — every transition driven by
// this method is the system, not a user; user-driven transitions
// flow through the Extractor use-cases with the user's identifier
// supplied by the HTTP handler.
const workerActor = "worker"

// readerTimeout caps the time spent inside the document reader
// composite — large XLSX or PDF files are the realistic worst-case
// (>1M cells / encrypted layers). The cap is per-document, not
// per-job; the surrounding river job timeout (5min) covers the
// other stages. ADR-016 §timeouts.
const readerTimeout = 10 * time.Second

func (w *ExtractionWorker) Process(ctx context.Context, args ExtractionArgs) error {
	ext, err := w.store.GetByID(ctx, args.ExtractionID)
	if err != nil {
		return fmt.Errorf("worker.Process load extraction %q: %w", args.ExtractionID, err)
	}
	if ext.Status != domain.StatusPending {
		return nil
	}
	if err := w.transition(ctx, ext, (*domain.Extraction).MarkProcessing, ""); err != nil {
		return fmt.Errorf("worker.Process transition pending->processing: %w", err)
	}

	data, filename, err := w.source.Fetch(ctx, ext.DocumentVersionID)
	if err != nil {
		return fmt.Errorf("worker.Process fetch document %q: %w", ext.DocumentVersionID, err)
	}

	parseCtx, cancel := context.WithTimeout(ctx, readerTimeout)
	defer cancel()
	_, err = w.reader.Parse(parseCtx, filename, data)
	if err != nil {
		return fmt.Errorf("worker.Process parse document %q: %w", filename, err)
	}

	return nil
}

// transition mutates the extraction via the supplied state-machine
// method, then records the audit event in a single UpdateStatus
// call so the post-mutation status and the audit trail are committed
// atomically by the repository. The transition method is passed as
// a Go method expression — callers write
// (*domain.Extraction).MarkProcessing rather than building a closure.
func (w *ExtractionWorker) transition(ctx context.Context, ext *domain.Extraction, mutate func(*domain.Extraction) error, errorMessage string) error {
	from := ext.Status
	if err := mutate(ext); err != nil {
		return fmt.Errorf("transition mutate from %s: %w", from, err)
	}
	event, err := domain.NewExtractionEvent(ext.ID, from, ext.Status, errorMessage, workerActor)
	if err != nil {
		return fmt.Errorf("transition build audit event %s->%s: %w", from, ext.Status, err)
	}
	if err := w.store.UpdateStatus(ctx, ext, event); err != nil {
		return fmt.Errorf("transition persist %s->%s: %w", from, ext.Status, err)
	}
	return nil
}
