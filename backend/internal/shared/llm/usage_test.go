// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

func TestNewTokenUsage_ComputesTotal(t *testing.T) {
	u := llm.NewTokenUsage(120, 80)
	if u.Prompt != 120 {
		t.Errorf("Prompt = %d, want 120", u.Prompt)
	}
	if u.Completion != 80 {
		t.Errorf("Completion = %d, want 80", u.Completion)
	}
	if u.Total != 200 {
		t.Errorf("Total = %d, want 200", u.Total)
	}
}

func TestNewTokenUsage_ZeroIsValid(t *testing.T) {
	u := llm.NewTokenUsage(0, 0)
	if u.Total != 0 {
		t.Errorf("zero usage: Total = %d, want 0", u.Total)
	}
}

func TestNewTokenUsage_ClampsNegativeInput(t *testing.T) {
	u := llm.NewTokenUsage(-5, -10)
	if u.Prompt != 0 || u.Completion != 0 || u.Total != 0 {
		t.Errorf("negative inputs not clamped: %+v", u)
	}
}

func TestZeroTokenUsage_ReturnsZeroValue(t *testing.T) {
	u := llm.ZeroTokenUsage()
	if u.Prompt != 0 || u.Completion != 0 || u.Total != 0 {
		t.Errorf("ZeroTokenUsage returned non-zero: %+v", u)
	}
}

func TestTokenUsage_Add(t *testing.T) {
	tests := []struct {
		name                                     string
		aPrompt, aCompletion, bPrompt, bCompletion int
		wantPrompt, wantCompletion, wantTotal     int
	}{
		{
			name:           "basic",
			aPrompt:        10,
			aCompletion:    5,
			bPrompt:        20,
			bCompletion:    7,
			wantPrompt:     30,
			wantCompletion: 12,
			wantTotal:      42,
		},
		{
			name:           "zero left identity",
			aPrompt:        0,
			aCompletion:    0,
			bPrompt:        15,
			bCompletion:    25,
			wantPrompt:     15,
			wantCompletion: 25,
			wantTotal:      40,
		},
		{
			name:           "zero right identity",
			aPrompt:        15,
			aCompletion:    25,
			bPrompt:        0,
			bCompletion:    0,
			wantPrompt:     15,
			wantCompletion: 25,
			wantTotal:      40,
		},
		{
			name:           "large numbers",
			aPrompt:        100,
			aCompletion:    200,
			bPrompt:        300,
			bCompletion:    400,
			wantPrompt:     400,
			wantCompletion: 600,
			wantTotal:      1000,
		},
		{
			name:           "asymmetric",
			aPrompt:        0,
			aCompletion:    5,
			bPrompt:        7,
			bCompletion:    0,
			wantPrompt:     7,
			wantCompletion: 5,
			wantTotal:      12,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := llm.NewTokenUsage(tc.aPrompt, tc.aCompletion)
			b := llm.NewTokenUsage(tc.bPrompt, tc.bCompletion)
			sum := a.Add(b)
			if sum.Prompt != tc.wantPrompt || sum.Completion != tc.wantCompletion || sum.Total != tc.wantTotal {
				t.Errorf("Add: %+v, want {%d %d %d}", sum, tc.wantPrompt, tc.wantCompletion, tc.wantTotal)
			}
		})
	}
}

func TestTokenUsage_AddDoesNotMutate(t *testing.T) {
	a := llm.NewTokenUsage(10, 5)
	b := llm.NewTokenUsage(20, 7)
	_ = a.Add(b)
	if a.Prompt != 10 || a.Completion != 5 || a.Total != 15 {
		t.Errorf("a mutated after Add: %+v, want {10 5 15}", a)
	}
	if b.Prompt != 20 || b.Completion != 7 || b.Total != 27 {
		t.Errorf("b mutated after Add: %+v, want {20 7 27}", b)
	}
}
