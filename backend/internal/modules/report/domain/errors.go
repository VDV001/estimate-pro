// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

// Sentinel errors. Callers match via errors.Is. ADR-014 forbids dead
// sentinels — each lands together with the consumer branch that
// returns it. Pair C ships these alongside Format.IsValid validator
// (Pair C wires the use-case path that returns them).
var (
	// ErrInvalidFormat surfaces when a caller hands the use case a
	// Format that fails IsValid — typically from an HTTP query
	// param or a bot intent param the user typed wrong.
	ErrInvalidFormat = errors.New("report: invalid format")

	// ErrEmptyEstimation surfaces when the project has no submitted
	// estimations to aggregate — the report cannot describe a
	// PERT view that has nothing in it. UI / bot consumers map
	// this to a user-facing "submit estimations first" message.
	ErrEmptyEstimation = errors.New("report: no submitted estimations to aggregate")
)
