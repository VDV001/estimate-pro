// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
)

// Extractor exposes the synchronous lifecycle operations of the
// extractor module. Worker-side processing (LLM extraction,
// document download, prompt-injection check) lives in the worker
// package and ships with PR-B3.
//
// maxBytes caps the file size accepted by RequestExtraction; a
// zero or negative value disables the guard, which is what unit
// tests want when the size path is not under examination.
//
// enqueuer dispatches an async river job after a new extraction is
// persisted (RequestExtraction) or re-queued (RetryExtraction). A nil
// value is accepted for unit tests that do not exercise the enqueue
// path; the composition root always supplies a real implementation.
type Extractor struct {
	repo     ExtractionRepository
	maxBytes int64
	enqueuer JobEnqueuer
}

func NewExtractor(repo ExtractionRepository, maxBytes int64, enqueuer JobEnqueuer) *Extractor {
	return &Extractor{repo: repo, maxBytes: maxBytes, enqueuer: enqueuer}
}

// RequestExtraction is the entry point for "extract tasks from this
// document version". It enforces three checks in order:
//
//   1. file size — fileSize > maxBytes (if maxBytes > 0) surfaces
//      ErrDocumentTooLarge before any repository write.
//   2. idempotency — if a pending / processing / completed
//      extraction already exists for the (documentID,
//      documentVersionID) pair, it is returned verbatim. The
//      caller can inspect the status to decide UX (continue
//      polling vs show existing tasks).
//   3. domain invariants — NewExtraction validates that both ids
//      are non-empty, surfacing ErrMissingDocument /
//      ErrMissingDocumentVersion otherwise.
//
// Only after those checks does Repo.Create run — the UNIQUE partial
// index in migration 009 is still the last line of defence against
// a race between the GetActive lookup and the Create.
//
// The actor argument is currently unused on the create path
// (the initial pending row has no prior status to audit) but is
// retained in the signature so future hooks (e.g. emitting a
// "requested" lifecycle event) can wire it without another
// breaking change.
func (u *Extractor) RequestExtraction(ctx context.Context, documentID, documentVersionID string, fileSize int64, actor string) (*domain.Extraction, error) {
	_ = actor // reserved for the lifecycle-event hook noted above

	if u.maxBytes > 0 && fileSize > u.maxBytes {
		return nil, fmt.Errorf("request: %w (size=%d, max=%d)", domain.ErrDocumentTooLarge, fileSize, u.maxBytes)
	}

	existing, err := u.repo.GetActiveByDocumentVersion(ctx, documentID, documentVersionID)
	switch {
	case err == nil:
		return existing, nil
	case errors.Is(err, domain.ErrExtractionNotFound):
		// fall through to create
	default:
		return nil, fmt.Errorf("request: lookup active: %w", err)
	}

	ext, err := domain.NewExtraction(documentID, documentVersionID)
	if err != nil {
		return nil, err
	}
	if err := u.repo.Create(ctx, ext); err != nil {
		return nil, fmt.Errorf("request: create: %w", err)
	}
	if u.enqueuer != nil {
		if err := u.enqueuer.Enqueue(ctx, ext.ID); err != nil {
			return nil, fmt.Errorf("request: enqueue: %w", err)
		}
	}
	return ext, nil
}

// CancelExtraction transitions an Extraction to cancelled and
// appends the corresponding audit event in a single repository
// call. ErrExtractionNotFound surfaces when no row exists;
// ErrInvalidStatusTransition surfaces when the extraction is
// already in a terminal state (completed/failed/cancelled).
func (u *Extractor) CancelExtraction(ctx context.Context, id, actor string) error {
	return u.transition(ctx, id, actor, "cancel", (*domain.Extraction).MarkCancelled)
}

// RetryExtraction re-arms a failed Extraction back to pending so the
// worker can pick it up again. Anything other than failed surfaces
// ErrInvalidStatusTransition; missing rows surface
// ErrExtractionNotFound.
func (u *Extractor) RetryExtraction(ctx context.Context, id, actor string) error {
	if err := u.transition(ctx, id, actor, "retry", (*domain.Extraction).MarkRetry); err != nil {
		return err
	}
	if u.enqueuer != nil {
		if err := u.enqueuer.Enqueue(ctx, id); err != nil {
			return fmt.Errorf("retry: enqueue: %w", err)
		}
	}
	return nil
}

// GetExtraction loads the aggregate and the full audit trail. Both
// errors surface verbatim (ErrExtractionNotFound from GetByID
// included) so handlers can map them directly.
func (u *Extractor) GetExtraction(ctx context.Context, id string) (*domain.Extraction, []*domain.ExtractionEvent, error) {
	ext, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	events, err := u.repo.GetEvents(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return ext, events, nil
}

// ListByProject forwards to the repository — a thin pass-through is
// acceptable because the only logic the handler needs is "give me
// every extraction for this project, newest first" and that maps
// 1:1 onto the persistence query.
func (u *Extractor) ListByProject(ctx context.Context, projectID string) ([]*domain.Extraction, error) {
	return u.repo.ListByProject(ctx, projectID)
}

// transition centralises the get → mutate → audit → save pattern
// shared by Cancel and Retry. The mutate argument is a method
// expression on *Extraction so each caller passes the appropriate
// state-machine method without duplicating the surrounding plumbing.
func (u *Extractor) transition(ctx context.Context, id, actor, op string, mutate func(*domain.Extraction) error) error {
	ext, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	fromStatus := ext.Status
	if err := mutate(ext); err != nil {
		return err
	}
	ev, err := domain.NewExtractionEvent(ext.ID, fromStatus, ext.Status, "", actor)
	if err != nil {
		return fmt.Errorf("%s: build audit event: %w", op, err)
	}
	return u.repo.UpdateStatus(ctx, ext, ev)
}
