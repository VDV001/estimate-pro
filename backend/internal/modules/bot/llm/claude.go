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

const defaultClaudeBaseURL = "https://api.anthropic.com"

// ClaudeParser implements domain.LLMParser using the Anthropic Claude API.
type ClaudeParser struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewClaudeParser creates a new ClaudeParser with the given API key and model.
func NewClaudeParser(apiKey, model string) *ClaudeParser {
	return &ClaudeParser{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultClaudeBaseURL,
		client:  &http.Client{},
	}
}

// claudeRequest is the request body for the Anthropic Messages API.
type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse is the response body from the Anthropic Messages API.
type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// ParseIntent parses a user message into a structured Intent using Claude.
func (p *ClaudeParser) ParseIntent(ctx context.Context, message string, history []string) (*domain.Intent, error) {
	slog.InfoContext(ctx, "ClaudeParser.ParseIntent: start", slog.String("model", p.model), slog.Int("msg_len", len(message)), slog.Int("history_len", len(history)))
	start := time.Now()
	userPrompt := BuildUserPrompt(message, history)

	reqBody := claudeRequest{
		Model:     p.model,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []claudeMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ClaudeParser.ParseIntent: marshal request: %w", err)
	}

	url := p.baseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("ClaudeParser.ParseIntent: create request: %w", err)
	}

	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "ClaudeParser.ParseIntent: HTTP request failed", slog.String("error", err.Error()), slog.Duration("elapsed", time.Since(start)))
		return nil, fmt.Errorf("ClaudeParser.ParseIntent: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ClaudeParser.ParseIntent: read response: %w", err)
	}

	slog.InfoContext(ctx, "ClaudeParser.ParseIntent: API responded", slog.Int("status", resp.StatusCode), slog.Int("body_len", len(respBody)), slog.Duration("elapsed", time.Since(start)))

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "ClaudeParser.ParseIntent: non-200 status", slog.Int("status", resp.StatusCode), slog.String("body", string(respBody)))
		return nil, fmt.Errorf("ClaudeParser.ParseIntent: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		slog.ErrorContext(ctx, "ClaudeParser.ParseIntent: unmarshal failed", slog.String("error", err.Error()), slog.String("body", string(respBody)))
		return nil, fmt.Errorf("ClaudeParser.ParseIntent: unmarshal response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		slog.ErrorContext(ctx, "ClaudeParser.ParseIntent: empty content in response")
		return nil, fmt.Errorf("ClaudeParser.ParseIntent: empty content in response")
	}

	slog.DebugContext(ctx, "ClaudeParser.ParseIntent: raw LLM output", slog.String("text", claudeResp.Content[0].Text))

	intent, err := parseIntentResponse([]byte(claudeResp.Content[0].Text))
	if err != nil {
		slog.ErrorContext(ctx, "ClaudeParser.ParseIntent: parse intent JSON failed", slog.String("error", err.Error()), slog.String("raw", claudeResp.Content[0].Text))
		return nil, fmt.Errorf("ClaudeParser.ParseIntent: %w", err)
	}

	slog.InfoContext(ctx, "ClaudeParser.ParseIntent: done", slog.String("intent", string(intent.Type)), slog.Float64("confidence", intent.Confidence), slog.Duration("elapsed", time.Since(start)))
	intent.RawText = message
	return intent, nil
}
