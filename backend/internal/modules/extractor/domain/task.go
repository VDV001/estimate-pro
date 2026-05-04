// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "strings"

// maxTaskNameLen mirrors document.maxDocumentTitleLen — 255 chars is
// the practical cap for human-readable task names extracted by the
// LLM and matches the storage column width planned for PR-B2's
// migration.
const maxTaskNameLen = 255

// ExtractedTask is a value object carrying one task surfaced by the
// LLM-based extractor. EstimateHint is whatever free-form duration or
// effort hint the LLM returned (e.g. "8h", "3 days") and is allowed
// to be empty.
type ExtractedTask struct {
	Name         string
	EstimateHint string
}

// NewExtractedTask constructs an ExtractedTask, trimming both fields
// and enforcing 1..maxTaskNameLen on the trimmed name. Returns
// ErrInvalidTaskName otherwise.
func NewExtractedTask(name, hint string) (ExtractedTask, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" || len(trimmedName) > maxTaskNameLen {
		return ExtractedTask{}, ErrInvalidTaskName
	}
	return ExtractedTask{
		Name:         trimmedName,
		EstimateHint: strings.TrimSpace(hint),
	}, nil
}
