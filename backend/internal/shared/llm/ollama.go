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
	ollamaAPIPath        = "/api/chat"
	defaultOllamaBaseURL = "http://localhost:11434"
)

// OllamaAdapter calls a local (or self-hosted) Ollama instance. No API
// key is required; the base URL is configurable. Implements both
// [Completer] and [IntentParser].
type OllamaAdapter struct {
	model   string
	baseURL string
	client  *http.Client
}

// NewOllamaAdapter constructs an OllamaAdapter with a default http.Client
// (60s per-request timeout). When baseURL is empty, [defaultOllamaBaseURL]
// is used.
func NewOllamaAdapter(model, baseURL string) *OllamaAdapter {
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	return &OllamaAdapter{
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// NewOllamaAdapterWithClient injects a custom http.Client (test seam).
func NewOllamaAdapterWithClient(model, baseURL string, client *http.Client) *OllamaAdapter {
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	return &OllamaAdapter{
		model:   model,
		baseURL: baseURL,
		client:  client,
	}
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Stream   bool            `json:"stream"`
	Messages []ollamaMessage `json:"messages"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	PromptEvalCount int `json:"prompt_eval_count"`
	EvalCount       int `json:"eval_count"`
}

// Complete implements [Completer] against the Ollama Chat API
// (`/api/chat` with stream=false).
func (a *OllamaAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string, opts CompletionOptions) (text string, usage TokenUsage, err error) {
	start := time.Now()
	_ = opts // Ollama ignores MaxTokens/Temperature/JSONMode here — model-side config

	bodyBytes, err := json.Marshal(ollamaRequest{
		Model:  a.model,
		Stream: false,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return "", ZeroTokenUsage(), fmt.Errorf("OllamaAdapter.Complete: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+ollamaAPIPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", ZeroTokenUsage(), fmt.Errorf("OllamaAdapter.Complete: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "OllamaAdapter.Complete: HTTP error",
			slog.String("error", err.Error()),
			slog.Duration("elapsed", time.Since(start)))
		return "", ZeroTokenUsage(), fmt.Errorf("OllamaAdapter.Complete: %w: %v", ErrLLMHTTP, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ZeroTokenUsage(), fmt.Errorf("OllamaAdapter.Complete: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "OllamaAdapter.Complete: non-200",
			slog.Int("status", resp.StatusCode),
			slog.String("body_preview", bodyPreview(respBody)),
			slog.Duration("elapsed", time.Since(start)))
		return "", ZeroTokenUsage(), fmt.Errorf("OllamaAdapter.Complete: %w: status %d", ErrLLMHTTP, resp.StatusCode)
	}

	var or ollamaResponse
	if err := json.Unmarshal(respBody, &or); err != nil {
		slog.ErrorContext(ctx, "OllamaAdapter.Complete: unmarshal failed",
			slog.String("error", err.Error()),
			slog.String("body_preview", bodyPreview(respBody)))
		return "", ZeroTokenUsage(), fmt.Errorf("OllamaAdapter.Complete: %w: %v", ErrLLMResponseInvalid, err)
	}

	usage = NewTokenUsage(or.PromptEvalCount, or.EvalCount)
	slog.InfoContext(ctx, "OllamaAdapter.Complete: ok",
		slog.String("model", a.model),
		slog.Int("status", resp.StatusCode),
		slog.Int("prompt_tokens", usage.Prompt),
		slog.Int("completion_tokens", usage.Completion),
		slog.Duration("elapsed", time.Since(start)))
	return or.Message.Content, usage, nil
}

// ParseIntent implements [IntentParser] by delegating to Complete with
// JSONMode hint. Ollama doesn't honour the hint server-side; the
// caller's prompt elicits JSON output.
func (a *OllamaAdapter) ParseIntent(ctx context.Context, systemPrompt, userPrompt string) (rawJSON string, usage TokenUsage, err error) {
	return a.Complete(ctx, systemPrompt, userPrompt, CompletionOptions{JSONMode: true})
}
