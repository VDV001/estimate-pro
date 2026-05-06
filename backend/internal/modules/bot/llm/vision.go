// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	visionAPIPath            = "/v1/messages"
	defaultVisionBaseURL     = "https://api.anthropic.com"
	visionAnthropicVersion   = "2023-06-01"
	visionMaxTokens          = 2048
	visionDefaultHTTPTimeout = 60 * time.Second
	visionUserPrompt         = "Извлеки весь текст с изображения. Верни только распознанный текст без комментариев и форматирования."
	visionDefaultMediaType   = "image/jpeg"
	visionBodyPreviewLen     = 200
)

// ErrVisionEmptyResponse signals that Claude returned a 200 OK with no
// content blocks. Treated as a recognition failure: callers in the bot
// usecase map it to a "не удалось распознать" user message instead of
// retrying — empty content usually means the upstream model refused or
// the picture had no extractable text.
var ErrVisionEmptyResponse = errors.New("ClaudeVisionAdapter: empty response content")

// ErrVisionHTTP signals a non-2xx HTTP status from the Anthropic
// endpoint. Wrapped with errors.Is/As-friendly fmt.Errorf chains so
// callers can branch on transport-level vs response-level failures.
var ErrVisionHTTP = errors.New("ClaudeVisionAdapter: HTTP error")

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

type visionImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type visionContentBlock struct {
	Type   string             `json:"type"`
	Text   string             `json:"text,omitempty"`
	Source *visionImageSource `json:"source,omitempty"`
}

type visionMessage struct {
	Role    string               `json:"role"`
	Content []visionContentBlock `json:"content"`
}

type visionRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []visionMessage `json:"messages"`
}

type visionResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// ExtractTextFromImage sends imageBytes to the Anthropic Messages API
// as a base64-encoded image content block paired with an OCR
// instruction, and returns the raw model output. The implementation
// hard-codes media_type to image/jpeg — Telegram photos arrive as
// JPEG, and Claude tolerates a small mismatch on PNG inputs without
// failing the request.
func (a *ClaudeVisionAdapter) ExtractTextFromImage(ctx context.Context, imageBytes []byte) (string, error) {
	start := time.Now()

	encoded := base64.StdEncoding.EncodeToString(imageBytes)
	bodyBytes, err := json.Marshal(visionRequest{
		Model:     a.model,
		MaxTokens: visionMaxTokens,
		Messages: []visionMessage{{
			Role: "user",
			Content: []visionContentBlock{
				{
					Type: "image",
					Source: &visionImageSource{
						Type:      "base64",
						MediaType: visionDefaultMediaType,
						Data:      encoded,
					},
				},
				{Type: "text", Text: visionUserPrompt},
			},
		}},
	})
	if err != nil {
		return "", fmt.Errorf("ClaudeVisionAdapter.ExtractTextFromImage: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+visionAPIPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("ClaudeVisionAdapter.ExtractTextFromImage: create request: %w", err)
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", visionAnthropicVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "ClaudeVisionAdapter.ExtractTextFromImage: HTTP error",
			slog.String("error", err.Error()),
			slog.Duration("elapsed", time.Since(start)))
		return "", fmt.Errorf("ClaudeVisionAdapter.ExtractTextFromImage: %w: %v", ErrVisionHTTP, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ClaudeVisionAdapter.ExtractTextFromImage: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "ClaudeVisionAdapter.ExtractTextFromImage: non-200",
			slog.Int("status", resp.StatusCode),
			slog.String("body_preview", visionBodyPreview(respBody)),
			slog.Duration("elapsed", time.Since(start)))
		return "", fmt.Errorf("ClaudeVisionAdapter.ExtractTextFromImage: %w: status %d", ErrVisionHTTP, resp.StatusCode)
	}

	var vr visionResponse
	if err := json.Unmarshal(respBody, &vr); err != nil {
		slog.ErrorContext(ctx, "ClaudeVisionAdapter.ExtractTextFromImage: unmarshal failed",
			slog.String("error", err.Error()),
			slog.String("body_preview", visionBodyPreview(respBody)))
		return "", fmt.Errorf("ClaudeVisionAdapter.ExtractTextFromImage: unmarshal: %w", err)
	}
	if len(vr.Content) == 0 {
		slog.WarnContext(ctx, "ClaudeVisionAdapter.ExtractTextFromImage: empty content",
			slog.String("body_preview", visionBodyPreview(respBody)))
		return "", ErrVisionEmptyResponse
	}

	slog.InfoContext(ctx, "ClaudeVisionAdapter.ExtractTextFromImage: ok",
		slog.String("model", a.model),
		slog.Int("status", resp.StatusCode),
		slog.Int("text_len", len(vr.Content[0].Text)),
		slog.Duration("elapsed", time.Since(start)))
	return vr.Content[0].Text, nil
}

// visionBodyPreview caps an HTTP response body to a short string for
// structured logs — provider error envelopes can echo prompts or keys
// in their content (cf. shared/llm.bodyPreview from PR #42).
func visionBodyPreview(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) > visionBodyPreviewLen {
		return string(b[:visionBodyPreviewLen])
	}
	return string(b)
}
