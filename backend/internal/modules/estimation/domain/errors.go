// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

var (
	ErrEstimationNotFound = errors.New("estimation not found")
	ErrAlreadySubmitted   = errors.New("estimation already submitted")
	ErrEmptyItems         = errors.New("estimation must have at least one item")
	ErrNotDraft           = errors.New("only draft estimations can be modified")
	ErrInvalidHours       = errors.New("hours must be non-negative and min <= likely <= max")
	ErrNotAuthor          = errors.New("only the author can perform this action")
	ErrTaskNameRequired   = errors.New("task name is required")
	ErrTaskNameTooLong    = errors.New("task name too long (max 255)")
	ErrEmptyInput         = errors.New("empty input")
	ErrNoValidItems       = errors.New("no valid items found")
	ErrMissingProject     = errors.New("estimation project is required")
	ErrMissingAuthor      = errors.New("estimation author is required")
)
