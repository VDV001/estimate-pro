// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrInvalidChannel       = errors.New("invalid notification channel")
	ErrDeliveryFailed       = errors.New("notification delivery failed")
	ErrMissingUser          = errors.New("notification user is required")
	ErrInvalidEventType     = errors.New("invalid notification event type")
	ErrMissingTitle         = errors.New("notification title is required")
)
