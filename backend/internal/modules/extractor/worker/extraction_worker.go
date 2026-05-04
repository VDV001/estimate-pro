// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package worker hosts the asynchronous body of the extractor
// pipeline. PR-B2 ships only the placeholder scaffold — the real
// Process implementation (MinIO download → shared/reader.Composite
// dispatch → security prompt-injection guard → sharedllm.Completer
// JSON-mode call → schema validation → SaveTasks + UpdateStatus
// audit trail) lands in PR-B3 (issue #7). River queue registration
// and retry policy are also deferred to that PR; the type shape and
// JSON-tagged ExtractionArgs are pinned now so river-side wiring is
// a drop-in change rather than a refactor.
package worker

import (
	"context"
	"errors"
)

// ErrWorkerNotImplemented is the sentinel returned by Process while
// the worker body is deferred. Callers detect the deferred state
// via errors.Is and surface a typed UX (e.g. the river runner can
// fail fast and not retry). Removed in PR-B3 alongside the real
// body — ADR-014 forbids dead sentinels, so this one ships with a
// real consumer (Process below) and exits the codebase together
// with that consumer.
var ErrWorkerNotImplemented = errors.New("extractor/worker: process not implemented (deferred to PR-B3, issue #7)")

// ExtractionArgs is the JSON payload enqueued onto the river queue
// for a single extraction job. The shape is fixed in PR-B2 so the
// queue contract is stable across the deferral boundary; PR-B3
// will register this type with river.AddWorker without changing
// the wire format.
type ExtractionArgs struct {
	ExtractionID string `json:"extraction_id"`
}

// ExtractionWorker is the river-compatible worker for processing
// extraction jobs. PR-B2 defines the type without dependencies;
// PR-B3 expands the constructor to accept the persistence port,
// the document storage adapter, the shared reader composite, the
// shared LLM completer, and the security checker. The expansion
// is a constructor signature change rather than a redesign — main.go
// already gates the extractor module behind FEATURE_DOCUMENT_PIPELINE_ENABLED
// so wiring is concentrated in one place.
type ExtractionWorker struct{}

// NewExtractionWorker returns a placeholder worker whose Process
// method is hard-wired to ErrWorkerNotImplemented. The real
// dependency-accepting constructor ships in PR-B3.
func NewExtractionWorker() *ExtractionWorker {
	return &ExtractionWorker{}
}

// Process is invoked by the river job runner for each enqueued
// ExtractionArgs. PR-B2 returns ErrWorkerNotImplemented so the
// pipeline is observably deferred; PR-B3 replaces this body with
// the orchestration described in the package doc.
func (w *ExtractionWorker) Process(ctx context.Context, args ExtractionArgs) error {
	_ = ctx
	_ = args
	return ErrWorkerNotImplemented
}
