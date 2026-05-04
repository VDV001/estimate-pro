// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

// formatterMaxTokens caps the formatter's output length. 512 is enough
// for a one-paragraph Esti reply without runaway costs.
const formatterMaxTokens = 512

// Formatter rephrases raw action results in Esti's voice via LLM #2.
// It never sees user input — only the action result string and the
// intent type. HTTP work is fully delegated to a shared/llm.Completer
// passed at construction; the formatter owns only the prompt assembly
// and the raw-fallback policy on completer error.
type Formatter struct {
	completer sharedllm.Completer
}

// NewFormatter wraps a shared completer. Passing nil is supported —
// the Formatter then short-circuits to the raw action result, which
// is the same fallback applied when the completer errors out. This
// lets the composition root degrade gracefully when env LLM config
// is missing or invalid.
func NewFormatter(completer sharedllm.Completer) *Formatter {
	return &Formatter{completer: completer}
}

// Format rephrases actionResult in Esti's personality. On any error
// from the completer (HTTP failure, malformed response, timeout) the
// raw actionResult is returned with nil error — formatting failure
// must not break the bot's reply flow.
func (f *Formatter) Format(ctx context.Context, actionResult string, intentType domain.IntentType) (string, error) {
	if f == nil || f.completer == nil {
		return actionResult, nil
	}

	start := time.Now()
	userPrompt := fmt.Sprintf("Действие: %s\nРезультат:\n%s", intentType, actionResult)
	slog.InfoContext(ctx, "Formatter.Format: start",
		slog.String("intent", string(intentType)),
		slog.Int("result_len", len(actionResult)))

	text, usage, err := f.completer.Complete(ctx, formatterPrompt, userPrompt, sharedllm.CompletionOptions{MaxTokens: formatterMaxTokens})
	if err != nil {
		slog.WarnContext(ctx, "Formatter.Format: completer error, returning raw result",
			slog.String("intent", string(intentType)),
			slog.String("error", err.Error()),
			slog.Duration("elapsed", time.Since(start)))
		return actionResult, nil //nolint:nilerr // raw fallback preserves UX
	}

	slog.InfoContext(ctx, "Formatter.Format: done",
		slog.Int("formatted_len", len(text)),
		slog.Int("tokens_total", usage.Total),
		slog.Duration("elapsed", time.Since(start)))
	return text, nil
}

// FormatReaction picks an appropriate emoji reaction for the intent type.
func FormatReaction(intentType domain.IntentType) string {
	switch intentType {
	case domain.IntentCreateProject:
		return "🚀"
	case domain.IntentAddMember:
		return "🎉"
	case domain.IntentSubmitEstimation:
		return "🔥"
	case domain.IntentUploadDocument:
		return "👀"
	case domain.IntentGetAggregated:
		return "📊"
	case domain.IntentListProjects, domain.IntentListMembers, domain.IntentGetProjectStatus:
		return "👍"
	case domain.IntentForgotPassword:
		return "🔑"
	case domain.IntentHelp:
		return "💡"
	default:
		return ""
	}
}

// MentionReaction returns a reaction for casual mentions (not commands).
func MentionReaction() string {
	return "👋"
}
