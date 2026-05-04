// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import "context"

// CompletionOptions tunes a single Complete call. Defaults (zero values)
// are sane for short-form structured-JSON extraction: 1024 max tokens,
// temperature unset (provider default), JSONMode left to caller's prompt
// to enforce ("respond ONLY with a JSON object").
type CompletionOptions struct {
	// MaxTokens caps the response length. Zero means adapter default
	// (currently 1024 across all providers).
	MaxTokens int

	// Temperature controls randomness. Zero means adapter default
	// (typically 0.7 — provider-specific).
	Temperature float64

	// JSONMode hints that the caller expects strict JSON output.
	// Adapters supporting a structured-output flag (OpenAI's
	// response_format: {type: json_object}) opt in; others rely on the
	// prompt itself to elicit JSON.
	JSONMode bool
}

// Completer is a generic structured-completion port. Returns the raw
// text reply from the provider plus a TokenUsage record (zero if
// unreported).
//
// Errors are typed (errors.Is sentinels): ErrLLMHTTP,
// ErrLLMResponseInvalid, ErrLLMTimeout — all defined in domain.go.
// Callers must not parse error strings.
//
// Implementations: ClaudeAdapter, OpenAIAdapter, OllamaAdapter (in this
// package). Bot module wraps a Completer in BotIntentParser for intent
// classification; future extractor module uses Completer directly for
// task extraction.
type Completer interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string, opts CompletionOptions) (text string, usage TokenUsage, err error)
}
