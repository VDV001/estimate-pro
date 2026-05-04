// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// ExtractionEvent is the audit-trail value object emitted by the
// repository on every state transition of an Extraction. Storing one
// row per transition (in extraction_events) lets operators
// reconstruct the full history of an extraction independently of the
// current Status field.
//
// Actor is a free-form string identifying who drove the transition:
// "worker" for the river job, "user:<uuid>" for explicit user
// action, "system" for automated cleanups.
type ExtractionEvent struct {
	ID           string
	ExtractionID string
	FromStatus   ExtractionStatus
	ToStatus     ExtractionStatus
	ErrorMessage string
	Actor        string
	CreatedAt    time.Time
}

// NewExtractionEvent constructs an ExtractionEvent enforcing
// invariants: non-empty trimmed extraction id, both statuses must
// satisfy ExtractionStatus.IsValid, non-empty trimmed actor.
// ErrorMessage is optional and trimmed.
func NewExtractionEvent(extractionID string, from, to ExtractionStatus, errorMessage, actor string) (*ExtractionEvent, error) {
	ext := strings.TrimSpace(extractionID)
	if ext == "" {
		return nil, ErrMissingExtraction
	}
	if !from.IsValid() || !to.IsValid() {
		return nil, ErrInvalidStatusTransition
	}
	act := strings.TrimSpace(actor)
	if act == "" {
		return nil, ErrInvalidActor
	}
	return &ExtractionEvent{
		ID:           uuid.New().String(),
		ExtractionID: ext,
		FromStatus:   from,
		ToStatus:     to,
		ErrorMessage: strings.TrimSpace(errorMessage),
		Actor:        act,
		CreatedAt:    time.Now(),
	}, nil
}
