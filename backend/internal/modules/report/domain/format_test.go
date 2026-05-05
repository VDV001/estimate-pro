// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/report/domain"
)

func TestFormat_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		format domain.Format
		want   bool
	}{
		{"md is valid", domain.FormatMD, true},
		{"pdf is valid", domain.FormatPDF, true},
		{"docx is valid", domain.FormatDOCX, true},
		{"empty is invalid", "", false},
		{"random is invalid", "yaml", false},
		{"upper-case is invalid (canonical lower-case)", "PDF", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.format.IsValid()
			if got != tc.want {
				t.Errorf("Format(%q).IsValid() = %v, want %v", tc.format, got, tc.want)
			}
		})
	}
}

func TestErrInvalidFormat_IsErrorsIs_friendly(t *testing.T) {
	wrapped := errors.Join(domain.ErrInvalidFormat, errors.New("query param: yaml"))
	if !errors.Is(wrapped, domain.ErrInvalidFormat) {
		t.Errorf("errors.Is(wrapped, ErrInvalidFormat) = false, want true")
	}
}

func TestErrEmptyEstimation_IsErrorsIs_friendly(t *testing.T) {
	wrapped := errors.Join(domain.ErrEmptyEstimation, errors.New("project p1 has 0 submitted estimations"))
	if !errors.Is(wrapped, domain.ErrEmptyEstimation) {
		t.Errorf("errors.Is(wrapped, ErrEmptyEstimation) = false, want true")
	}
}
