// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package llm provides shared LLM provider infrastructure: HTTP adapters,
// completion API, intent-classification API, and token usage tracking.
//
// Two interfaces are exposed: IntentParser (raw JSON return for bot intent
// classification) and Completer (generic structured-completion for arbitrary
// callers like extractor). One adapter struct per provider implements both.
package llm

// TokenUsage records prompt + completion token counts reported by the
// provider. Total is recomputed (Prompt+Completion) at construction so it
// remains consistent even when providers omit the total field.
//
// Providers that do not report usage (Ollama in some configurations) yield
// ZeroTokenUsage — callers must not assume non-zero values.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// NewTokenUsage constructs a TokenUsage with consistent total.
func NewTokenUsage(prompt, completion int) TokenUsage {
	return TokenUsage{
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      prompt + completion,
	}
}

// ZeroTokenUsage is the explicit zero value, used when a provider omits
// usage data or a call short-circuits before reaching the provider.
func ZeroTokenUsage() TokenUsage { return TokenUsage{} }

// Add combines two usage records — used to aggregate usage across multiple
// LLM calls within a single business operation (e.g. classifier + formatter).
func (u TokenUsage) Add(other TokenUsage) TokenUsage {
	return NewTokenUsage(
		u.PromptTokens+other.PromptTokens,
		u.CompletionTokens+other.CompletionTokens,
	)
}
