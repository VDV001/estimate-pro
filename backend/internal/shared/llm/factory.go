// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import "fmt"

// Parser is the union interface that bot intent parsing and generic
// completion consumers depend on. All four adapters in this package
// satisfy it (Grok shares OpenAIAdapter via the x.ai base URL).
type Parser interface {
	Completer
	IntentParser
}

// defaultGrokBaseURL is the x.ai endpoint Grok routes through. The
// factory passes it to OpenAIAdapter when provider is [ProviderGrok]
// and the caller did not override baseURL.
const defaultGrokBaseURL = "https://api.x.ai"

// NewParser constructs a provider-specific adapter implementing Parser
// (both Completer and IntentParser). The composition root and tests use
// it; production code should not instantiate adapters directly.
//
// Validation:
//   - provider must be one of [ProviderClaude], [ProviderOpenAI],
//     [ProviderGrok], [ProviderOllama]; otherwise [ErrInvalidProvider].
//   - model must be non-empty; otherwise [ErrEmptyModel].
//   - apiKey must be non-empty for Claude/OpenAI/Grok; Ollama is exempt
//     (local provider). Otherwise [ErrEmptyAPIKey].
//
// baseURL overrides the provider default when non-empty (e.g. tests
// pointing at httptest.Server). Grok uses [defaultGrokBaseURL] when
// baseURL is empty.
func NewParser(provider LLMProviderType, apiKey, model, baseURL string) (Parser, error) {
	if !provider.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidProvider, string(provider))
	}
	if model == "" {
		return nil, ErrEmptyModel
	}
	switch provider {
	case ProviderClaude:
		if apiKey == "" {
			return nil, fmt.Errorf("%w: provider %s", ErrEmptyAPIKey, provider)
		}
		return NewClaudeAdapter(apiKey, model, baseURL), nil
	case ProviderOpenAI:
		if apiKey == "" {
			return nil, fmt.Errorf("%w: provider %s", ErrEmptyAPIKey, provider)
		}
		return NewOpenAIAdapter(apiKey, model, baseURL), nil
	case ProviderGrok:
		if apiKey == "" {
			return nil, fmt.Errorf("%w: provider %s", ErrEmptyAPIKey, provider)
		}
		grokURL := baseURL
		if grokURL == "" {
			grokURL = defaultGrokBaseURL
		}
		return NewOpenAIAdapter(apiKey, model, grokURL), nil
	case ProviderOllama:
		return NewOllamaAdapter(model, baseURL), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidProvider, string(provider))
	}
}
