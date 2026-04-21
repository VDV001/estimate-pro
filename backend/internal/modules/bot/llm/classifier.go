// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"context"
	"log/slog"
	"strings"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

// MentionType classifies how the bot was mentioned in a group chat.
type MentionType string

const (
	MentionDirected  MentionType = "directed"  // User is asking the bot to do something.
	MentionCasual    MentionType = "mentioned" // Bot name mentioned but no request.
	MentionUnrelated MentionType = "unrelated" // Not related to the bot at all.
)

// Classifier uses LLM #1 to determine if a group message is directed at the bot.
// Its system prompt contains NO personality info — nothing to steal.
type Classifier struct {
	parser domain.LLMParser
}

// NewClassifier creates a Classifier backed by an LLM parser.
func NewClassifier(parser domain.LLMParser) *Classifier {
	return &Classifier{parser: parser}
}

// Classify determines if a message is directed at the bot, a casual mention, or unrelated.
func (c *Classifier) Classify(ctx context.Context, message string) MentionType {
	slog.DebugContext(ctx, "Classifier.Classify: start", slog.Int("msg_len", len(message)))

	// Use the classifier prompt (no personality) via ParseIntent.
	// We temporarily override by passing the classifier prompt as the message
	// with a special prefix the parser will handle.
	intent, err := c.parser.ParseIntent(ctx, message, []string{"__classifier__"})
	if err != nil {
		// On error, assume directed (fail open for UX).
		slog.WarnContext(ctx, "Classifier.Classify: LLM failed, assuming directed", slog.String("error", err.Error()))
		return MentionDirected
	}

	// The classifier returns the classification in intent.Type.
	result := strings.ToLower(strings.TrimSpace(intent.RawText))
	if result == "" {
		result = strings.ToLower(string(intent.Type))
	}

	var mentionType MentionType
	switch {
	case strings.Contains(result, "directed"):
		mentionType = MentionDirected
	case strings.Contains(result, "mentioned"):
		mentionType = MentionCasual
	case strings.Contains(result, "unrelated"):
		mentionType = MentionUnrelated
	default:
		mentionType = MentionDirected // fail open
	}

	slog.DebugContext(ctx, "Classifier.Classify: done", slog.String("result", string(mentionType)))
	return mentionType
}
