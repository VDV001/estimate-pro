// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package domain holds the Extractor module's domain model: the Extraction
// aggregate with its state machine, value objects (ExtractedTask), and
// sentinel errors. The package depends only on stdlib + uuid; persistence,
// HTTP, LLM, and worker concerns live in sibling packages.
package domain

// ExtractionStatus is a value object covering every state in the
// Extraction state machine:
//
//	pending ─────→ processing ──┬──→ completed
//	   │                ↑       └──→ failed ──→ pending (via MarkRetry)
//	   └────────────────┴───────→ cancelled
type ExtractionStatus string

const (
	StatusPending    ExtractionStatus = "pending"
	StatusProcessing ExtractionStatus = "processing"
	StatusCompleted  ExtractionStatus = "completed"
	StatusFailed     ExtractionStatus = "failed"
	StatusCancelled  ExtractionStatus = "cancelled"
)

func (s ExtractionStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusProcessing, StatusCompleted, StatusFailed, StatusCancelled:
		return true
	}
	return false
}

func (s ExtractionStatus) String() string { return string(s) }
