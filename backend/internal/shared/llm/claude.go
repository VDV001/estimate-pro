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
)

const (
	claudeAPIPath        = "/v1/messages"
	defaultClaudeBaseURL = "https://api.anthropic.com"
	claudeAnthropicVer   = "2023-06-01"
	defaultMaxTokens     = 1024
)

// ClaudeAdapter calls the Anthropic Messages API. Implements both
// Completer (generic structured-completion) and IntentParser (raw JSON
// return for bot intent classification).
type ClaudeAdapter struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewClaudeAdapter constructs a ClaudeAdapter with a default http.Client
// (60s per-request timeout). Use NewClaudeAdapterWithClient in tests to
// inject a custom http.RoundTripper.
func NewClaudeAdapter(apiKey, model, baseURL string) *ClaudeAdapter {
	if baseURL == "" {
		baseURL = defaultClaudeBaseURL
	}
	return &ClaudeAdapter{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// NewClaudeAdapterWithClient injects a custom http.Client (test seam).
// Production code should call NewClaudeAdapter.
func NewClaudeAdapterWithClient(apiKey, model, baseURL string, client *http.Client) *ClaudeAdapter {
	if baseURL == "" {
		baseURL = defaultClaudeBaseURL
	}
	return &ClaudeAdapter{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  client,
	}
}

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

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Complete implements [Completer] against the Anthropic Messages API.
func (a *ClaudeAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string, opts CompletionOptions) (text string, usage TokenUsage, err error) {
	start := time.Now()
	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	bodyBytes, err := json.Marshal(claudeRequest{
		Model:     a.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  []claudeMessage{{Role: "user", Content: userPrompt}},
	})
	if err != nil {
		return "", ZeroTokenUsage(), fmt.Errorf("ClaudeAdapter.Complete: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+claudeAPIPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", ZeroTokenUsage(), fmt.Errorf("ClaudeAdapter.Complete: create request: %w", err)
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", claudeAnthropicVer)
	req.Header.Set("content-type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "ClaudeAdapter.Complete: HTTP error",
			slog.String("error", err.Error()),
			slog.Duration("elapsed", time.Since(start)))
		return "", ZeroTokenUsage(), fmt.Errorf("ClaudeAdapter.Complete: %w: %v", ErrLLMHTTP, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ZeroTokenUsage(), fmt.Errorf("ClaudeAdapter.Complete: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "ClaudeAdapter.Complete: non-200",
			slog.Int("status", resp.StatusCode),
			slog.String("body_preview", bodyPreview(respBody)),
			slog.Duration("elapsed", time.Since(start)))
		return "", ZeroTokenUsage(), fmt.Errorf("ClaudeAdapter.Complete: %w: status %d", ErrLLMHTTP, resp.StatusCode)
	}

	var cr claudeResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		slog.ErrorContext(ctx, "ClaudeAdapter.Complete: unmarshal failed",
			slog.String("error", err.Error()),
			slog.String("body_preview", bodyPreview(respBody)))
		return "", ZeroTokenUsage(), fmt.Errorf("ClaudeAdapter.Complete: %w: %v", ErrLLMResponseInvalid, err)
	}
	if len(cr.Content) == 0 {
		slog.WarnContext(ctx, "ClaudeAdapter.Complete: empty content",
			slog.String("body_preview", bodyPreview(respBody)))
		return "", ZeroTokenUsage(), fmt.Errorf("ClaudeAdapter.Complete: %w: empty content", ErrLLMResponseInvalid)
	}

	usage = NewTokenUsage(cr.Usage.InputTokens, cr.Usage.OutputTokens)
	slog.InfoContext(ctx, "ClaudeAdapter.Complete: ok",
		slog.String("model", a.model),
		slog.Int("status", resp.StatusCode),
		slog.Int("prompt_tokens", usage.Prompt),
		slog.Int("completion_tokens", usage.Completion),
		slog.Duration("elapsed", time.Since(start)))
	return cr.Content[0].Text, usage, nil
}

// ParseIntent implements [IntentParser] by delegating to Complete with
// JSONMode hint. Caller is expected to have prepared a JSON-eliciting
// system prompt; the returned text is raw JSON for the caller to
// unmarshal into its own Intent type.
func (a *ClaudeAdapter) ParseIntent(ctx context.Context, systemPrompt, userPrompt string) (rawJSON string, usage TokenUsage, err error) {
	return a.Complete(ctx, systemPrompt, userPrompt, CompletionOptions{JSONMode: true})
}
