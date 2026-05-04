// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"fmt"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
)

// Extractor exposes the synchronous lifecycle operations of the
// extractor module. Worker-side processing (LLM extraction,
// document download, prompt-injection check) lives in the worker
// package and ships with PR-B3.
type Extractor struct {
	repo ExtractionRepository
}

func NewExtractor(repo ExtractionRepository) *Extractor {
	return &Extractor{repo: repo}
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
	return u.transition(ctx, id, actor, "retry", (*domain.Extraction).MarkRetry)
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
