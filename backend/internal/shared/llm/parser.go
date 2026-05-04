// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import "context"

// IntentParser is the bot-oriented narrowing of Completer: same low-level
// HTTP call, but the API contract documents intent-classification
// semantics. Caller passes a system prompt that elicits structured-JSON
// intent output; parser returns the raw JSON for the caller to unmarshal
// into its own domain Intent type. Token usage is reported.
//
// Why a separate interface (vs. just using Completer): keeps shared/llm
// free of bot domain types — the Intent type lives in bot/domain. Bot's
// adapter wraps a Completer with bot-specific prompt assembly and
// JSON-extraction logic.
//
// Implementations: same adapters as Completer (single struct per provider
// satisfies both interfaces). The factory NewParser (defined in Task 11)
// returns a union Parser interface = Completer + IntentParser.
type IntentParser interface {
	ParseIntent(ctx context.Context, systemPrompt, userPrompt string) (rawJSON string, usage TokenUsage, err error)
}
