// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"log/slog"
	"math/rand/v2"
	"strings"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/llm"
)

// isPromptInjection checks if the message looks like a prompt injection attempt.
func isPromptInjection(text string) bool {
	lower := strings.ToLower(text)
	for _, pattern := range llm.InputFilterPatterns() {
		if strings.Contains(lower, pattern) {
			slog.Warn("isPromptInjection: matched", slog.String("pattern", pattern), slog.Int("text_len", len(text)))
			return true
		}
	}
	return false
}

// deflectionResponse returns a random playful deflection.
func deflectionResponse() string {
	responses := llm.InjectionDeflections()
	return responses[rand.IntN(len(responses))]
}
