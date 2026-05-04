// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

// BotIntentParser adapts a shared [sharedllm.IntentParser] into the
// bot-specific [domain.LLMParser]: it owns the bot's system prompt,
// builds the user prompt from conversation history, unmarshals the raw
// JSON returned by the provider into [*domain.Intent], and attaches the
// original user message as RawText for downstream logging.
//
// Why a facade rather than a direct interface alias: the shared parser
// returns raw JSON + token usage, while bot callers need a structured
// Intent. Putting the JSON-into-Intent step here keeps shared/llm free of
// bot domain types.
type BotIntentParser struct {
	inner sharedllm.IntentParser
}

// NewBotIntentParser wraps a shared adapter so it satisfies bot's
// [domain.LLMParser] contract.
func NewBotIntentParser(inner sharedllm.IntentParser) *BotIntentParser {
	return &BotIntentParser{inner: inner}
}

// ParseIntent satisfies [domain.LLMParser]. It assembles the user prompt
// (history + current message), delegates the HTTP/JSON exchange to the
// inner shared adapter, then unmarshals the JSON envelope and attaches
// the original message as RawText.
//
// Errors from the inner adapter are wrapped with %w so callers can still
// match shared sentinels via errors.Is (ErrLLMHTTP, ErrLLMResponseInvalid,
// ErrLLMTimeout).
func (b *BotIntentParser) ParseIntent(ctx context.Context, message string, history []string) (*domain.Intent, error) {
	userPrompt := BuildUserPrompt(message, history)
	raw, usage, err := b.inner.ParseIntent(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("BotIntentParser.ParseIntent: %w", err)
	}
	slog.DebugContext(ctx, "BotIntentParser.ParseIntent: raw output",
		slog.Int("raw_len", len(raw)),
		slog.Int("tokens_total", usage.Total))
	intent, err := parseIntentResponse([]byte(raw))
	if err != nil {
		return nil, fmt.Errorf("BotIntentParser.ParseIntent: %w", err)
	}
	intent.RawText = message
	return intent, nil
}

// Compile-time check.
var _ domain.LLMParser = (*BotIntentParser)(nil)
