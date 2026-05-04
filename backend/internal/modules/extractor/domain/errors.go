// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

// Sentinel errors. Callers match via errors.Is. ADR-014 forbids dead
// sentinels — each new sentinel lands together with the consumer
// branch that returns it, never speculatively.
var (
	ErrInvalidTaskName         = errors.New("extractor: invalid task name")
	ErrMissingDocument         = errors.New("extractor: document id is required")
	ErrMissingDocumentVersion  = errors.New("extractor: document version id is required")
	ErrInvalidStatusTransition = errors.New("extractor: invalid status transition")
	ErrAlreadyCompleted        = errors.New("extractor: extraction already completed")
	ErrMissingExtraction       = errors.New("extractor: extraction id is required")
	ErrInvalidActor            = errors.New("extractor: actor is required")
	ErrExtractionNotFound      = errors.New("extractor: extraction not found")
	ErrDocumentTooLarge        = errors.New("extractor: document exceeds maximum allowed size")
	ErrPromptInjectionDetected = errors.New("extractor: prompt injection detected in document text")
	ErrLLMResponseSchemaInvalid = errors.New("extractor: LLM response is not parseable JSON or violates the {tasks:[{name,estimate_hint}]} schema")
)
