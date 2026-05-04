// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import "time"

// bodyPreviewLen caps how many bytes of an HTTP response body we put
// into structured logs. Provider error envelopes can echo API keys or
// prompts in their content (lesson from PR #42 — OpenAI test output
// included the api_key in error responses). 200 bytes is enough to
// identify the failure class without bulk leak.
const bodyPreviewLen = 200

// bodyPreview returns up to bodyPreviewLen bytes of b as a string.
// Empty input yields the empty string. Used by adapter implementations
// in this package to log slog attrs without leaking large response
// bodies that may contain sensitive data.
func bodyPreview(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) > bodyPreviewLen {
		return string(b[:bodyPreviewLen])
	}
	return string(b)
}

// defaultHTTPTimeout is the per-request timeout for provider calls in
// adapter HTTP clients. Existing parsers had no timeout (a goroutine
// leak risk on hung connections). 60s comfortably covers Anthropic /
// OpenAI typical response time (5-15s) plus headroom for long prompts.
//
// Caller-overridable via adapter constructors that accept a custom
// http.Client (test seam — see NewClaudeAdapterWithClient and friends
// in Tasks 7-10).
const defaultHTTPTimeout = 60 * time.Second
