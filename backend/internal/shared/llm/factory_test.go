// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

func TestNewParser_Claude_ReturnsClaudeAdapter(t *testing.T) {
	p, err := llm.NewParser(llm.ProviderClaude, "key", "claude-3", "")
	if err != nil {
		t.Fatalf("NewParser(Claude): %v", err)
	}
	if _, ok := p.(*llm.ClaudeAdapter); !ok {
		t.Errorf("NewParser(Claude) = %T, want *ClaudeAdapter", p)
	}
}

func TestNewParser_OpenAI_ReturnsOpenAIAdapter(t *testing.T) {
	p, err := llm.NewParser(llm.ProviderOpenAI, "key", "gpt-4", "")
	if err != nil {
		t.Fatalf("NewParser(OpenAI): %v", err)
	}
	if _, ok := p.(*llm.OpenAIAdapter); !ok {
		t.Errorf("NewParser(OpenAI) = %T, want *OpenAIAdapter", p)
	}
}

func TestNewParser_Grok_ReturnsOpenAICompatibleAdapter(t *testing.T) {
	p, err := llm.NewParser(llm.ProviderGrok, "key", "grok-2", "")
	if err != nil {
		t.Fatalf("NewParser(Grok): %v", err)
	}
	if _, ok := p.(*llm.OpenAIAdapter); !ok {
		t.Errorf("NewParser(Grok) = %T, want *OpenAIAdapter (Grok routes here)", p)
	}
}

func TestNewParser_Ollama_NoAPIKeyRequired(t *testing.T) {
	p, err := llm.NewParser(llm.ProviderOllama, "", "llama-3", "")
	if err != nil {
		t.Fatalf("NewParser(Ollama, empty key): %v", err)
	}
	if _, ok := p.(*llm.OllamaAdapter); !ok {
		t.Errorf("NewParser(Ollama) = %T, want *OllamaAdapter", p)
	}
}

func TestNewParser_InvalidProvider_ReturnsErrInvalidProvider(t *testing.T) {
	_, err := llm.NewParser(llm.LLMProviderType("bogus"), "key", "model", "")
	if !errors.Is(err, llm.ErrInvalidProvider) {
		t.Errorf("error = %v, want errors.Is(ErrInvalidProvider)", err)
	}
}

func TestNewParser_EmptyModel_ReturnsErrEmptyModel(t *testing.T) {
	_, err := llm.NewParser(llm.ProviderClaude, "key", "", "")
	if !errors.Is(err, llm.ErrEmptyModel) {
		t.Errorf("error = %v, want errors.Is(ErrEmptyModel)", err)
	}
}

func TestNewParser_EmptyAPIKey_NonOllamaReturnsErrEmptyAPIKey(t *testing.T) {
	for _, prov := range []llm.LLMProviderType{llm.ProviderClaude, llm.ProviderOpenAI, llm.ProviderGrok} {
		t.Run(string(prov), func(t *testing.T) {
			_, err := llm.NewParser(prov, "", "model", "")
			if !errors.Is(err, llm.ErrEmptyAPIKey) {
				t.Errorf("provider=%s error = %v, want errors.Is(ErrEmptyAPIKey)", prov, err)
			}
		})
	}
}

func TestNewParser_ResultSatisfiesParserInterface(t *testing.T) {
	p, err := llm.NewParser(llm.ProviderClaude, "key", "claude-3", "")
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	var _ llm.Parser = p
	var _ llm.Completer = p
	var _ llm.IntentParser = p
}
