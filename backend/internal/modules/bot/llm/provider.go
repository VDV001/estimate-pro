// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

// NewParser creates a bot-domain LLMParser by composing a shared adapter
// with the bot's prompt-building + intent-JSON-parsing logic
// ([BotIntentParser]). Provider type and base URL semantics defer to
// [sharedllm.NewParser].
//
// Errors propagate the shared sentinels [sharedllm.ErrInvalidProvider],
// [sharedllm.ErrEmptyModel], [sharedllm.ErrEmptyAPIKey] — callers detect
// via errors.Is.
func NewParser(provider domain.LLMProviderType, apiKey, model, baseURL string) (domain.LLMParser, error) {
	inner, err := sharedllm.NewParser(provider, apiKey, model, baseURL)
	if err != nil {
		return nil, err
	}
	return NewBotIntentParser(inner), nil
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
