// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

// Sentinel errors. Callers match via errors.Is. ADR-014 forbids dead
// sentinels — each new sentinel lands together with the consumer
// branch that returns it, never speculatively. Future PR-B2/B3
// sentinels (ErrInvalidStatusTransition, ErrAlreadyCompleted,
// ErrMissingDocument, ErrCancelled, ErrExtractionNotFound,
// ErrLLMResponseSchemaInvalid, ErrPromptInjectionDetected,
// ErrDocumentTooLarge) join this list as their use-cases ship.
var (
	ErrInvalidTaskName        = errors.New("extractor: invalid task name")
	ErrMissingDocument        = errors.New("extractor: document id is required")
	ErrMissingDocumentVersion = errors.New("extractor: document version id is required")
)
