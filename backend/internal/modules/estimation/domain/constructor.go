// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxTaskNameLen = 255

// NewEstimationItem constructs an item enforcing domain invariants:
// non-empty trimmed name up to 255 chars, non-negative hours, min <= likely <= max
// (equality allowed for fixed-estimate tasks). ID is auto-generated. EstimationID
// and SortOrder are set later via AttachTo when the item is linked to an estimation.
func NewEstimationItem(taskName string, min, likely, max float64, note string) (*EstimationItem, error) {
	trimmed := strings.TrimSpace(taskName)
	if trimmed == "" {
		return nil, ErrTaskNameRequired
	}
	if len(trimmed) > maxTaskNameLen {
		return nil, ErrTaskNameTooLong
	}
	if min < 0 || likely < 0 || max < 0 {
		return nil, ErrInvalidHours
	}
	if min > likely || likely > max {
		return nil, ErrInvalidHours
	}
	return &EstimationItem{
		ID:          uuid.New().String(),
		TaskName:    trimmed,
		MinHours:    min,
		LikelyHours: likely,
		MaxHours:    max,
		Note:        note,
	}, nil
}

// AttachTo links the item to an estimation by setting EstimationID and SortOrder.
// Intended to be called by the persistence layer when items are saved in batch.
func (item *EstimationItem) AttachTo(estimationID string, sortOrder int) {
	item.EstimationID = estimationID
	item.SortOrder = sortOrder
}

// NewEstimation constructs an Estimation enforcing invariants: non-empty project
// and author. ID is auto-generated. Status defaults to draft, CreatedAt=now,
// SubmittedAt=zero until Submit is called.
func NewEstimation(projectID, submittedBy, documentVersionID string) (*Estimation, error) {
	if projectID == "" {
		return nil, ErrMissingProject
	}
	if submittedBy == "" {
		return nil, ErrMissingAuthor
	}
	return &Estimation{
		ID:                uuid.New().String(),
		ProjectID:         projectID,
		DocumentVersionID: documentVersionID,
		SubmittedBy:       submittedBy,
		Status:            StatusDraft,
		CreatedAt:         time.Now(),
	}, nil
}

// Submit marks the estimation as submitted and stamps SubmittedAt.
// Returns ErrAlreadySubmitted if called twice (idempotency guard).
func (e *Estimation) Submit() error {
	if e.IsSubmitted() {
		return ErrAlreadySubmitted
	}
	e.Status = StatusSubmitted
	e.SubmittedAt = time.Now()
	return nil
}

// AuthorizeAuthor reports whether the given userID is the estimation's author.
// Returns ErrNotAuthor otherwise.
func (e *Estimation) AuthorizeAuthor(userID string) error {
	if e.SubmittedBy != userID {
		return ErrNotAuthor
	}
	return nil
}
