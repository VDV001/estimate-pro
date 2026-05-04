// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

func TestNewTokenUsage_ComputesTotal(t *testing.T) {
	u := llm.NewTokenUsage(120, 80)
	if u.PromptTokens != 120 {
		t.Errorf("PromptTokens = %d, want 120", u.PromptTokens)
	}
	if u.CompletionTokens != 80 {
		t.Errorf("CompletionTokens = %d, want 80", u.CompletionTokens)
	}
	if u.TotalTokens != 200 {
		t.Errorf("TotalTokens = %d, want 200", u.TotalTokens)
	}
}

func TestNewTokenUsage_ZeroIsValid(t *testing.T) {
	u := llm.NewTokenUsage(0, 0)
	if u.TotalTokens != 0 {
		t.Errorf("zero usage: TotalTokens = %d, want 0", u.TotalTokens)
	}
}

func TestZeroTokenUsage_ReturnsZeroValue(t *testing.T) {
	u := llm.ZeroTokenUsage()
	if u.PromptTokens != 0 || u.CompletionTokens != 0 || u.TotalTokens != 0 {
		t.Errorf("ZeroTokenUsage returned non-zero: %+v", u)
	}
}

func TestTokenUsage_Add(t *testing.T) {
	a := llm.NewTokenUsage(10, 5)
	b := llm.NewTokenUsage(20, 7)
	sum := a.Add(b)
	if sum.PromptTokens != 30 || sum.CompletionTokens != 12 || sum.TotalTokens != 42 {
		t.Errorf("Add: %+v, want {30 12 42}", sum)
	}
}
