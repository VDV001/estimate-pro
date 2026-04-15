// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"context"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

// mockLLMParser implements domain.LLMParser for testing.
type mockLLMParser struct {
	parseIntentFn func(ctx context.Context, message string, history []string) (*domain.Intent, error)
}

func (m *mockLLMParser) ParseIntent(ctx context.Context, message string, history []string) (*domain.Intent, error) {
	if m.parseIntentFn != nil {
		return m.parseIntentFn(ctx, message, history)
	}
	return &domain.Intent{Type: domain.IntentUnknown}, nil
}

func TestClassifier_Classify(t *testing.T) {
	tests := []struct {
		name    string
		message string
		intent  *domain.Intent
		err     error
		want    MentionType
	}{
		{
			name:    "directed via RawText",
			message: "Эсти, покажи проекты",
			intent:  &domain.Intent{Type: domain.IntentListProjects, RawText: "directed"},
			want:    MentionDirected,
		},
		{
			name:    "mentioned via RawText",
			message: "Эсти красавчик",
			intent:  &domain.Intent{Type: domain.IntentUnknown, RawText: "mentioned"},
			want:    MentionCasual,
		},
		{
			name:    "unrelated via RawText",
			message: "скинь файл эстимейта",
			intent:  &domain.Intent{Type: domain.IntentUnknown, RawText: "unrelated"},
			want:    MentionUnrelated,
		},
		{
			name:    "directed via intent type when RawText empty",
			message: "create project",
			intent:  &domain.Intent{Type: "directed", RawText: ""},
			want:    MentionDirected,
		},
		{
			name:    "mentioned via intent type when RawText empty",
			message: "Esti cool",
			intent:  &domain.Intent{Type: "mentioned", RawText: ""},
			want:    MentionCasual,
		},
		{
			name:    "unrelated via intent type when RawText empty",
			message: "random stuff",
			intent:  &domain.Intent{Type: "unrelated", RawText: ""},
			want:    MentionUnrelated,
		},
		{
			name:    "unknown result defaults to directed (fail open)",
			message: "something weird",
			intent:  &domain.Intent{Type: "gibberish", RawText: "gibberish"},
			want:    MentionDirected,
		},
		{
			name:    "LLM error defaults to directed (fail open)",
			message: "any message",
			err:     domain.ErrUnsupportedProvider,
			want:    MentionDirected,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parser := &mockLLMParser{
				parseIntentFn: func(_ context.Context, _ string, _ []string) (*domain.Intent, error) {
					if tc.err != nil {
						return nil, tc.err
					}
					return tc.intent, nil
				},
			}
			c := NewClassifier(parser)
			got := c.Classify(t.Context(), tc.message)
			if got != tc.want {
				t.Errorf("Classify(%q) = %q, want %q", tc.message, got, tc.want)
			}
		})
	}
}

func TestClassifier_Classify_PassesClassifierHistory(t *testing.T) {
	var gotHistory []string
	parser := &mockLLMParser{
		parseIntentFn: func(_ context.Context, _ string, history []string) (*domain.Intent, error) {
			gotHistory = history
			return &domain.Intent{Type: "directed", RawText: "directed"}, nil
		},
	}
	c := NewClassifier(parser)
	c.Classify(t.Context(), "test message")

	if len(gotHistory) != 1 || gotHistory[0] != "__classifier__" {
		t.Errorf("expected history [__classifier__], got %v", gotHistory)
	}
}

func TestInputFilterPatterns_NotEmpty(t *testing.T) {
	patterns := InputFilterPatterns()
	if len(patterns) == 0 {
		t.Fatal("InputFilterPatterns should return non-empty slice")
	}
	// Verify well-known patterns.
	wantPatterns := []string{
		"ignore previous",
		"system prompt",
		"jailbreak",
		"забудь всё",
		"покажи промпт",
	}
	set := make(map[string]bool, len(patterns))
	for _, p := range patterns {
		set[p] = true
	}
	for _, w := range wantPatterns {
		if !set[w] {
			t.Errorf("expected pattern %q in InputFilterPatterns", w)
		}
	}
}

func TestInjectionDeflections_NotEmpty(t *testing.T) {
	deflections := InjectionDeflections()
	if len(deflections) == 0 {
		t.Fatal("InjectionDeflections should return non-empty slice")
	}
	// Each deflection should be a non-empty string.
	for i, d := range deflections {
		if d == "" {
			t.Errorf("deflection[%d] is empty", i)
		}
	}
}
