// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
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
