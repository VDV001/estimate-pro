// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

// NewParser creates an LLMParser for the given provider type.
func NewParser(provider domain.LLMProviderType, apiKey, model, baseURL string) (domain.LLMParser, error) {
	switch provider {
	case domain.ProviderClaude:
		return NewClaudeParser(apiKey, model), nil
	case domain.ProviderOpenAI:
		return NewOpenAIParser(apiKey, model), nil
	case domain.ProviderGrok:
		return NewGrokParser(apiKey, model), nil
	case domain.ProviderOllama:
		return NewOllamaParser(baseURL, model), nil
	default:
		return nil, fmt.Errorf("NewParser: %w: %s", domain.ErrUnsupportedProvider, provider)
	}
}

// parseIntentResponse extracts JSON from the LLM response text and unmarshals it into an Intent.
// It handles cases where the LLM wraps the JSON in markdown code blocks.
func parseIntentResponse(body []byte) (*domain.Intent, error) {
	text := strings.TrimSpace(string(body))

	// Strip markdown code block wrappers if present.
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		// Remove first line (```json or ```) and last line (```)
		start := 1
		end := len(lines) - 1
		if end > start {
			text = strings.TrimSpace(strings.Join(lines[start:end], "\n"))
		}
	}

	var intent domain.Intent
	if err := json.Unmarshal([]byte(text), &intent); err != nil {
		return nil, fmt.Errorf("parseIntentResponse: %w", err)
	}

	if intent.Params == nil {
		intent.Params = make(map[string]string)
	}

	return &intent, nil
}
