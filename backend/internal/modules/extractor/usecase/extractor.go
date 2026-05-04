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
	ext, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	fromStatus := ext.Status
	if err := ext.MarkCancelled(); err != nil {
		return err
	}
	ev, err := domain.NewExtractionEvent(ext.ID, fromStatus, ext.Status, "", actor)
	if err != nil {
		return fmt.Errorf("cancel: build audit event: %w", err)
	}
	return u.repo.UpdateStatus(ctx, ext, ev)
}
