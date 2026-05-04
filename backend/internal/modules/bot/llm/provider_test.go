// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

// NewParser smoke tests — actual HTTP behaviour now lives in
// shared/llm/{claude,openai,ollama}_test.go. Here we only verify the bot
// facade wraps each shared adapter in a *BotIntentParser and propagates
// the shared validation sentinels.

func TestNewParser_AllProviders(t *testing.T) {
	tests := []struct {
		name     string
		provider domain.LLMProviderType
		apiKey   string
		baseURL  string
	}{
		{"claude", domain.ProviderClaude, "test-key", ""},
		{"openai", domain.ProviderOpenAI, "test-key", ""},
		{"grok", domain.ProviderGrok, "test-key", ""},
		{"ollama", domain.ProviderOllama, "", "http://localhost:11434"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.provider, tt.apiKey, "test-model", tt.baseURL)
			if err != nil {
				t.Fatalf("NewParser(%s) returned error: %v", tt.provider, err)
			}
			if parser == nil {
				t.Fatalf("NewParser(%s) returned nil parser", tt.provider)
			}
			if _, ok := parser.(*BotIntentParser); !ok {
				t.Errorf("NewParser(%s) returned %T, want *BotIntentParser", tt.provider, parser)
			}
		})
	}
}

func TestNewParser_PropagatesValidationSentinels(t *testing.T) {
	tests := []struct {
		name     string
		provider domain.LLMProviderType
		apiKey   string
		model    string
		wantErr  error
	}{
		{
			name:     "unsupported_provider",
			provider: "unsupported",
			apiKey:   "k",
			model:    "m",
			wantErr:  sharedllm.ErrInvalidProvider,
		},
		{
			name:     "empty_model",
			provider: domain.ProviderClaude,
			apiKey:   "k",
			model:    "",
			wantErr:  sharedllm.ErrEmptyModel,
		},
		{
			name:     "empty_api_key_for_claude",
			provider: domain.ProviderClaude,
			apiKey:   "",
			model:    "m",
			wantErr:  sharedllm.ErrEmptyAPIKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewParser(tt.provider, tt.apiKey, tt.model, "")
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want errors.Is(%v)", err, tt.wantErr)
			}
		})
	}
}

func TestParseIntentResponse_WithMarkdownWrapper(t *testing.T) {
	input := "```json\n{\"type\":\"list_projects\",\"params\":{},\"confidence\":0.9}\n```"

	intent, err := parseIntentResponse([]byte(input))
	if err != nil {
		t.Fatalf("parseIntentResponse returned error: %v", err)
	}

	if intent.Type != domain.IntentListProjects {
		t.Errorf("expected type=%s, got %s", domain.IntentListProjects, intent.Type)
	}
	if intent.Confidence != 0.9 {
		t.Errorf("expected confidence=0.9, got %f", intent.Confidence)
	}
}

func TestParseIntentResponse_InvalidJSON(t *testing.T) {
	_, err := parseIntentResponse([]byte("this is not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBuildUserPrompt_WithoutHistory(t *testing.T) {
	result := BuildUserPrompt("hello", nil)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestBuildUserPrompt_WithHistory(t *testing.T) {
	result := BuildUserPrompt("current message", []string{"prev1", "prev2"})
	if result == "current message" {
		t.Error("expected history to be included in prompt")
	}
}
