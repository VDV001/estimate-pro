// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"context"
	"errors"
	"net/http"
	"time"
)

const (
	visionAPIPath           = "/v1/messages"
	defaultVisionBaseURL    = "https://api.anthropic.com"
	visionAnthropicVersion  = "2023-06-01"
	visionMaxTokens         = 2048
	visionDefaultHTTPTimeout = 60 * time.Second
	visionUserPrompt        = "Извлеки весь текст с изображения. Верни только распознанный текст без комментариев и форматирования."
)

// ErrVisionNotImplemented is returned by ClaudeVisionAdapter while the
// adapter is still the RED-step stub. Removed once the GREEN
// implementation lands (TDD pair 1).
var ErrVisionNotImplemented = errors.New("ClaudeVisionAdapter: not implemented")

// ClaudeVisionAdapter calls the Anthropic Messages API with a
// multipart user message that combines a base64-encoded image and a
// text instruction asking Claude to OCR the picture. Implements
// usecase.TextExtractor — declared in bot/usecase/ports.go and wired
// into BotUsecase via the composition root.
type ClaudeVisionAdapter struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewClaudeVisionAdapter constructs a vision adapter with the default
// 60s HTTP timeout. baseURL falls back to the Anthropic production
// endpoint when empty so callers can pass "" in production wiring.
func NewClaudeVisionAdapter(apiKey, model, baseURL string) *ClaudeVisionAdapter {
	if baseURL == "" {
		baseURL = defaultVisionBaseURL
	}
	return &ClaudeVisionAdapter{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: visionDefaultHTTPTimeout},
	}
}

// NewClaudeVisionAdapterWithClient is the test seam — lets unit tests
// inject an httptest.Server-backed client without spinning up real
// network calls.
func NewClaudeVisionAdapterWithClient(apiKey, model, baseURL string, client *http.Client) *ClaudeVisionAdapter {
	if baseURL == "" {
		baseURL = defaultVisionBaseURL
	}
	return &ClaudeVisionAdapter{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  client,
	}
}

// ExtractTextFromImage RED stub — see ErrVisionNotImplemented. The
// GREEN-step commit replaces this with a real Claude Vision call.
func (a *ClaudeVisionAdapter) ExtractTextFromImage(ctx context.Context, imageBytes []byte) (string, error) {
	_ = ctx
	_ = imageBytes
	return "", ErrVisionNotImplemented
}
