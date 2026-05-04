// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
)

func TestExtractionStatus_IsValid(t *testing.T) {
	tests := []struct {
		name string
		s    domain.ExtractionStatus
		want bool
	}{
		{name: "pending", s: domain.StatusPending, want: true},
		{name: "processing", s: domain.StatusProcessing, want: true},
		{name: "completed", s: domain.StatusCompleted, want: true},
		{name: "failed", s: domain.StatusFailed, want: true},
		{name: "cancelled", s: domain.StatusCancelled, want: true},
		{name: "empty", s: "", want: false},
		{name: "garbage", s: "running", want: false},
		{name: "uppercase", s: "PENDING", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.IsValid(); got != tt.want {
				t.Errorf("ExtractionStatus(%q).IsValid()=%v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestExtractionStatus_String(t *testing.T) {
	tests := []struct {
		s    domain.ExtractionStatus
		want string
	}{
		{domain.StatusPending, "pending"},
		{domain.StatusProcessing, "processing"},
		{domain.StatusCompleted, "completed"},
		{domain.StatusFailed, "failed"},
		{domain.StatusCancelled, "cancelled"},
	}
	for _, tt := range tests {
		t.Run(string(tt.s), func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
