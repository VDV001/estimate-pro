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
	Prompt     int
	Completion int
	Total      int
}

// NewTokenUsage constructs a [TokenUsage] with consistent total. Recomputes
// Total from inputs to defend against providers that omit or miscount the
// total field. Negative inputs are clamped to 0 since token counts cannot
// be negative.
func NewTokenUsage(prompt, completion int) TokenUsage {
	if prompt < 0 {
		prompt = 0
	}
	if completion < 0 {
		completion = 0
	}
	return TokenUsage{
		Prompt:     prompt,
		Completion: completion,
		Total:      prompt + completion,
	}
}

// ZeroTokenUsage is the explicit zero value, used when a provider omits
// usage data or a call short-circuits before reaching the provider.
//
// Prefer `TokenUsage{}` in struct literals; use this factory when an
// explicit call reads better at the call site or for symmetry with
// [NewTokenUsage].
func ZeroTokenUsage() TokenUsage { return TokenUsage{} }

// Add combines two usage records — used to aggregate usage across multiple
// LLM calls within a single business operation (e.g. classifier + formatter).
func (u TokenUsage) Add(other TokenUsage) TokenUsage {
	return NewTokenUsage(
		u.Prompt+other.Prompt,
		u.Completion+other.Completion,
	)
}
