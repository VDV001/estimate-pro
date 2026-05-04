// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Extraction is the aggregate root for an LLM-driven task extraction
// run against a specific document version. State transitions are
// driven by methods on this type and follow the diagram in the
// status.go package comment. A nil StartedAt / CompletedAt means
// the corresponding transition has not happened yet.
type Extraction struct {
	ID                string
	DocumentID        string
	DocumentVersionID string
	Status            ExtractionStatus
	Tasks             []ExtractedTask
	FailureReason     string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	StartedAt         *time.Time
	CompletedAt       *time.Time
}

// NewExtraction constructs a pending Extraction enforcing non-empty
// document and version identifiers. Both fields are trimmed before
// validation. Auto-assigns ID, CreatedAt, UpdatedAt.
func NewExtraction(documentID, documentVersionID string) (*Extraction, error) {
	doc := strings.TrimSpace(documentID)
	if doc == "" {
		return nil, ErrMissingDocument
	}
	ver := strings.TrimSpace(documentVersionID)
	if ver == "" {
		return nil, ErrMissingDocumentVersion
	}
	now := time.Now()
	return &Extraction{
		ID:                uuid.New().String(),
		DocumentID:        doc,
		DocumentVersionID: ver,
		Status:            StatusPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// MarkProcessing transitions a pending Extraction into processing,
// stamping StartedAt. Any other starting status returns
// ErrInvalidStatusTransition with the offending status preserved.
func (e *Extraction) MarkProcessing() error {
	if e.Status != StatusPending {
		return fmt.Errorf("MarkProcessing from %s: %w", e.Status, ErrInvalidStatusTransition)
	}
	now := time.Now()
	e.Status = StatusProcessing
	e.StartedAt = &now
	e.UpdatedAt = now
	return nil
}

// MarkCompleted transitions a processing Extraction into completed
// with the supplied tasks. Re-completing an already-completed
// Extraction returns the more specific ErrAlreadyCompleted; every
// other rejected starting status returns ErrInvalidStatusTransition.
func (e *Extraction) MarkCompleted(tasks []ExtractedTask) error {
	if e.Status == StatusCompleted {
		return fmt.Errorf("MarkCompleted: %w", ErrAlreadyCompleted)
	}
	if e.Status != StatusProcessing {
		return fmt.Errorf("MarkCompleted from %s: %w", e.Status, ErrInvalidStatusTransition)
	}
	now := time.Now()
	e.Status = StatusCompleted
	e.Tasks = tasks
	e.CompletedAt = &now
	e.UpdatedAt = now
	return nil
}

// MarkFailed transitions a processing Extraction into failed and
// records the failure reason for downstream observability + retry
// UX. Re-failing or failing a non-processing Extraction returns
// ErrInvalidStatusTransition.
func (e *Extraction) MarkFailed(reason string) error {
	if e.Status != StatusProcessing {
		return fmt.Errorf("MarkFailed from %s: %w", e.Status, ErrInvalidStatusTransition)
	}
	now := time.Now()
	e.Status = StatusFailed
	e.FailureReason = reason
	e.CompletedAt = &now
	e.UpdatedAt = now
	return nil
}

// MarkCancelled transitions a pending or in-flight processing
// Extraction into cancelled. Once an Extraction is completed,
// failed, or already cancelled, it is terminal and the call returns
// ErrInvalidStatusTransition (no idempotent re-cancel — the caller
// must check status before requesting cancellation).
func (e *Extraction) MarkCancelled() error {
	if e.Status != StatusPending && e.Status != StatusProcessing {
		return fmt.Errorf("MarkCancelled from %s: %w", e.Status, ErrInvalidStatusTransition)
	}
	now := time.Now()
	e.Status = StatusCancelled
	e.CompletedAt = &now
	e.UpdatedAt = now
	return nil
}

// MarkRetry resurrects a failed Extraction back to pending so the
// worker can pick it up again. FailureReason / StartedAt /
// CompletedAt are cleared so the next cycle starts from a clean
// slate. Any non-failed starting status returns
// ErrInvalidStatusTransition.
func (e *Extraction) MarkRetry() error {
	if e.Status != StatusFailed {
		return fmt.Errorf("MarkRetry from %s: %w", e.Status, ErrInvalidStatusTransition)
	}
	e.Status = StatusPending
	e.FailureReason = ""
	e.StartedAt = nil
	e.CompletedAt = nil
	e.UpdatedAt = time.Now()
	return nil
}
