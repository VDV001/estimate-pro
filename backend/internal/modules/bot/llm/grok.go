// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

const defaultGrokBaseURL = "https://api.x.ai"

// GrokParser implements domain.LLMParser using the xAI Grok API (OpenAI-compatible).
type GrokParser struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewGrokParser creates a new GrokParser with the given API key and model.
// Default model is "grok-3-mini".
func NewGrokParser(apiKey, model string) *GrokParser {
	if model == "" {
		model = "grok-3-mini"
	}
	return &GrokParser{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultGrokBaseURL,
		client:  &http.Client{},
	}
}

// ParseIntent parses a user message into a structured Intent using Grok.
func (p *GrokParser) ParseIntent(ctx context.Context, message string, history []string) (*domain.Intent, error) {
	slog.InfoContext(ctx, "GrokParser.ParseIntent: start", slog.String("model", p.model), slog.Int("msg_len", len(message)), slog.Int("history_len", len(history)))
	start := time.Now()
	userPrompt := BuildUserPrompt(message, history)

	reqBody := openaiRequest{
		Model:     p.model,
		MaxTokens: 1024,
		Messages: []openaiMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("GrokParser.ParseIntent: marshal request: %w", err)
	}

	url := p.baseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("GrokParser.ParseIntent: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "GrokParser.ParseIntent: HTTP request failed", slog.String("error", err.Error()), slog.Duration("elapsed", time.Since(start)))
		return nil, fmt.Errorf("GrokParser.ParseIntent: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GrokParser.ParseIntent: read response: %w", err)
	}

	slog.InfoContext(ctx, "GrokParser.ParseIntent: API responded", slog.Int("status", resp.StatusCode), slog.Int("body_len", len(respBody)), slog.Duration("elapsed", time.Since(start)))

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "GrokParser.ParseIntent: non-200 status", slog.Int("status", resp.StatusCode), slog.String("body", string(respBody)))
		return nil, fmt.Errorf("GrokParser.ParseIntent: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var grokResp openaiResponse
	if err := json.Unmarshal(respBody, &grokResp); err != nil {
		slog.ErrorContext(ctx, "GrokParser.ParseIntent: unmarshal failed", slog.String("error", err.Error()), slog.String("body", string(respBody)))
		return nil, fmt.Errorf("GrokParser.ParseIntent: unmarshal response: %w", err)
	}

	if len(grokResp.Choices) == 0 {
		slog.ErrorContext(ctx, "GrokParser.ParseIntent: no choices in response")
		return nil, fmt.Errorf("GrokParser.ParseIntent: no choices in response")
	}

	slog.DebugContext(ctx, "GrokParser.ParseIntent: raw LLM output", slog.String("text", grokResp.Choices[0].Message.Content))

	intent, err := parseIntentResponse([]byte(grokResp.Choices[0].Message.Content))
	if err != nil {
		slog.ErrorContext(ctx, "GrokParser.ParseIntent: parse intent JSON failed", slog.String("error", err.Error()), slog.String("raw", grokResp.Choices[0].Message.Content))
		return nil, fmt.Errorf("GrokParser.ParseIntent: %w", err)
	}

	slog.InfoContext(ctx, "GrokParser.ParseIntent: done", slog.String("intent", string(intent.Type)), slog.Float64("confidence", intent.Confidence), slog.Duration("elapsed", time.Since(start)))
	intent.RawText = message
	return intent, nil
}
