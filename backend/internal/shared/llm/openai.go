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
	openaiAPIPath        = "/v1/chat/completions"
	defaultOpenAIBaseURL = "https://api.openai.com"
)

// OpenAIAdapter calls the OpenAI Chat Completions API. The same struct
// serves Grok via its OpenAI-compatible endpoint — the factory passes
// the Grok base URL when constructing for [ProviderGrok].
type OpenAIAdapter struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewOpenAIAdapter constructs an OpenAIAdapter with a default http.Client
// (60s per-request timeout). When baseURL is empty, [defaultOpenAIBaseURL]
// is used; pass an explicit Grok URL when constructing for Grok.
func NewOpenAIAdapter(apiKey, model, baseURL string) *OpenAIAdapter {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &OpenAIAdapter{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// NewOpenAIAdapterWithClient injects a custom http.Client (test seam).
func NewOpenAIAdapterWithClient(apiKey, model, baseURL string, client *http.Client) *OpenAIAdapter {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &OpenAIAdapter{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  client,
	}
}

type openaiRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []openaiMessage `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Complete implements [Completer] against the OpenAI Chat Completions API.
func (a *OpenAIAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string, opts CompletionOptions) (text string, usage TokenUsage, err error) {
	start := time.Now()
	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	bodyBytes, err := json.Marshal(openaiRequest{
		Model:     a.model,
		MaxTokens: maxTokens,
		Messages: []openaiMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return "", ZeroTokenUsage(), fmt.Errorf("OpenAIAdapter.Complete: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+openaiAPIPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", ZeroTokenUsage(), fmt.Errorf("OpenAIAdapter.Complete: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "OpenAIAdapter.Complete: HTTP error",
			slog.String("error", err.Error()),
			slog.Duration("elapsed", time.Since(start)))
		return "", ZeroTokenUsage(), fmt.Errorf("OpenAIAdapter.Complete: %w: %v", ErrLLMHTTP, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ZeroTokenUsage(), fmt.Errorf("OpenAIAdapter.Complete: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "OpenAIAdapter.Complete: non-200",
			slog.Int("status", resp.StatusCode),
			slog.String("body_preview", bodyPreview(respBody)),
			slog.Duration("elapsed", time.Since(start)))
		return "", ZeroTokenUsage(), fmt.Errorf("OpenAIAdapter.Complete: %w: status %d", ErrLLMHTTP, resp.StatusCode)
	}

	var or openaiResponse
	if err := json.Unmarshal(respBody, &or); err != nil {
		slog.ErrorContext(ctx, "OpenAIAdapter.Complete: unmarshal failed",
			slog.String("error", err.Error()),
			slog.String("body_preview", bodyPreview(respBody)))
		return "", ZeroTokenUsage(), fmt.Errorf("OpenAIAdapter.Complete: %w: %v", ErrLLMResponseInvalid, err)
	}
	if len(or.Choices) == 0 {
		slog.WarnContext(ctx, "OpenAIAdapter.Complete: no choices",
			slog.String("body_preview", bodyPreview(respBody)))
		return "", ZeroTokenUsage(), fmt.Errorf("OpenAIAdapter.Complete: %w: no choices", ErrLLMResponseInvalid)
	}

	usage = NewTokenUsage(or.Usage.PromptTokens, or.Usage.CompletionTokens)
	slog.InfoContext(ctx, "OpenAIAdapter.Complete: ok",
		slog.String("model", a.model),
		slog.Int("status", resp.StatusCode),
		slog.Int("prompt_tokens", usage.Prompt),
		slog.Int("completion_tokens", usage.Completion),
		slog.Duration("elapsed", time.Since(start)))
	return or.Choices[0].Message.Content, usage, nil
}

// ParseIntent implements [IntentParser] by delegating to Complete with
// JSONMode hint.
func (a *OpenAIAdapter) ParseIntent(ctx context.Context, systemPrompt, userPrompt string) (rawJSON string, usage TokenUsage, err error) {
	return a.Complete(ctx, systemPrompt, userPrompt, CompletionOptions{JSONMode: true})
}
