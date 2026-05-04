// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

func TestLLMProviderType_IsValid(t *testing.T) {
	cases := []struct {
		name  string
		input llm.LLMProviderType
		want  bool
	}{
		{"claude", llm.ProviderClaude, true},
		{"openai", llm.ProviderOpenAI, true},
		{"grok", llm.ProviderGrok, true},
		{"ollama", llm.ProviderOllama, true},
		{"unknown", llm.LLMProviderType("anthropic-v2"), false},
		{"empty", llm.LLMProviderType(""), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.input.IsValid(); got != c.want {
				t.Errorf("IsValid() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestNewLLMConfig_AcceptsSystemConfig(t *testing.T) {
	cfg, err := llm.NewLLMConfig("", llm.ProviderClaude, "sk-test", "claude-sonnet-4-6", "")
	if err != nil {
		t.Fatalf("system config: unexpected error %v", err)
	}
	if !cfg.IsSystem() {
		t.Error("expected IsSystem() == true for empty userID")
	}
}

func TestNewLLMConfig_AcceptsUserScopedConfig(t *testing.T) {
	cfg, err := llm.NewLLMConfig("user-1", llm.ProviderOpenAI, "sk-test", "gpt-4", "")
	if err != nil {
		t.Fatalf("user config: unexpected error %v", err)
	}
	if cfg.IsSystem() {
		t.Error("expected IsSystem() == false for non-empty userID")
	}
}

func TestNewLLMConfig_RejectsInvalidProvider(t *testing.T) {
	_, err := llm.NewLLMConfig("user-1", llm.LLMProviderType("invalid"), "key", "model", "")
	if !errors.Is(err, llm.ErrInvalidProvider) {
		t.Errorf("expected ErrInvalidProvider, got %v", err)
	}
}

func TestNewLLMConfig_RejectsEmptyModel(t *testing.T) {
	_, err := llm.NewLLMConfig("user-1", llm.ProviderClaude, "key", "", "")
	if !errors.Is(err, llm.ErrEmptyModel) {
		t.Errorf("expected ErrEmptyModel, got %v", err)
	}
}

func TestNewLLMConfig_RejectsEmptyAPIKeyForRemoteProviders(t *testing.T) {
	cases := []struct {
		name     string
		provider llm.LLMProviderType
	}{
		{"claude", llm.ProviderClaude},
		{"openai", llm.ProviderOpenAI},
		{"grok", llm.ProviderGrok},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := llm.NewLLMConfig("user-1", c.provider, "", "model", "")
			if !errors.Is(err, llm.ErrEmptyAPIKey) {
				t.Errorf("expected ErrEmptyAPIKey, got %v", err)
			}
		})
	}
}

func TestNewLLMConfig_AllowsEmptyAPIKeyForOllama(t *testing.T) {
	_, err := llm.NewLLMConfig("user-1", llm.ProviderOllama, "", "llama3", "")
	if err != nil {
		t.Errorf("Ollama with empty API key should succeed, got %v", err)
	}
}

func TestNewLLMConfig_TimestampsSet(t *testing.T) {
	cfg, err := llm.NewLLMConfig("u", llm.ProviderClaude, "k", "m", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if cfg.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
}

func TestLLMConfig_StringRedactsAPIKey(t *testing.T) {
	cfg, err := llm.NewLLMConfig("user-1", llm.ProviderClaude, "sk-secret-key-do-not-leak", "claude-test", "https://api.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := cfg.String()
	if !strings.Contains(s, "[REDACTED]") {
		t.Errorf("String() did not contain [REDACTED]: %s", s)
	}
	if strings.Contains(s, "sk-secret-key-do-not-leak") {
		t.Errorf("String() leaked API key: %s", s)
	}
}

func TestLLMConfig_StringHandlesNil(t *testing.T) {
	var cfg *llm.LLMConfig
	if got := cfg.String(); got != "<nil LLMConfig>" {
		t.Errorf("nil String() = %q, want <nil LLMConfig>", got)
	}
}
