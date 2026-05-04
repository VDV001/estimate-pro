// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
)

func TestNewExtractedTask(t *testing.T) {
	tests := []struct {
		name        string
		taskName    string
		hint        string
		wantErrIs   error
		wantName    string
		wantHint    string
	}{
		{name: "valid name and hint", taskName: "Design auth module", hint: "8h", wantName: "Design auth module", wantHint: "8h"},
		{name: "valid name without hint", taskName: "Implement login", hint: "", wantName: "Implement login", wantHint: ""},
		{name: "trims whitespace from name", taskName: "  Refactor parser  ", hint: "  3 days  ", wantName: "Refactor parser", wantHint: "3 days"},
		{name: "empty name", taskName: "", wantErrIs: domain.ErrInvalidTaskName},
		{name: "whitespace-only name", taskName: "   \t  ", wantErrIs: domain.ErrInvalidTaskName},
		{name: "name exceeds max length", taskName: strings.Repeat("a", 256), wantErrIs: domain.ErrInvalidTaskName},
		{name: "name exactly at max length", taskName: strings.Repeat("a", 255), wantName: strings.Repeat("a", 255)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := domain.NewExtractedTask(tt.taskName, tt.hint)
			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("err=%v, want errors.Is %v", err, tt.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if task.Name != tt.wantName {
				t.Errorf("Name=%q, want %q", task.Name, tt.wantName)
			}
			if task.EstimateHint != tt.wantHint {
				t.Errorf("EstimateHint=%q, want %q", task.EstimateHint, tt.wantHint)
			}
		})
	}
}
