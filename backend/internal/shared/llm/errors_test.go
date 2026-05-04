// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

func TestErrLLMHTTP_WrappedDetectableViaErrorsIs(t *testing.T) {
	wrapped := fmt.Errorf("provider call: %w", llm.ErrLLMHTTP)
	if !errors.Is(wrapped, llm.ErrLLMHTTP) {
		t.Error("errors.Is failed for wrapped ErrLLMHTTP")
	}
}

func TestErrLLMResponseInvalid_WrappedDetectableViaErrorsIs(t *testing.T) {
	wrapped := fmt.Errorf("decode: %w", llm.ErrLLMResponseInvalid)
	if !errors.Is(wrapped, llm.ErrLLMResponseInvalid) {
		t.Error("errors.Is failed for wrapped ErrLLMResponseInvalid")
	}
}

func TestErrLLMTimeout_WrappedDetectableViaErrorsIs(t *testing.T) {
	wrapped := fmt.Errorf("provider timeout: %w", llm.ErrLLMTimeout)
	if !errors.Is(wrapped, llm.ErrLLMTimeout) {
		t.Error("errors.Is failed for wrapped ErrLLMTimeout")
	}
}

func TestSentinelsAreDistinct(t *testing.T) {
	// Different sentinels must not satisfy errors.Is for each other —
	// otherwise callers cannot disambiguate failure classes.
	if errors.Is(llm.ErrLLMHTTP, llm.ErrLLMResponseInvalid) {
		t.Error("ErrLLMHTTP must be distinct from ErrLLMResponseInvalid")
	}
	if errors.Is(llm.ErrLLMHTTP, llm.ErrLLMTimeout) {
		t.Error("ErrLLMHTTP must be distinct from ErrLLMTimeout")
	}
	if errors.Is(llm.ErrLLMResponseInvalid, llm.ErrLLMTimeout) {
		t.Error("ErrLLMResponseInvalid must be distinct from ErrLLMTimeout")
	}
}
