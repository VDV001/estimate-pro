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

// Formatter uses a separate LLM call (LLM #2) to rephrase action results
// in Esti's voice. It NEVER receives user input — only the action result.
type Formatter struct {
	provider domain.LLMProviderType
	apiKey   string
	model    string
	baseURL  string
	client   *http.Client
}

// NewFormatter creates a Formatter for the given LLM provider.
func NewFormatter(provider domain.LLMProviderType, apiKey, model, baseURL string) *Formatter {
	return &Formatter{
		provider: provider,
		apiKey:   apiKey,
		model:    model,
		baseURL:  baseURL,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Format rephrases a raw action result in Esti's personality.
func (f *Formatter) Format(ctx context.Context, actionResult string, intentType domain.IntentType) (string, error) {
	slog.InfoContext(ctx, "Formatter.Format: start", slog.String("provider", string(f.provider)), slog.String("intent", string(intentType)), slog.Int("result_len", len(actionResult)))
	start := time.Now()
	userPrompt := fmt.Sprintf("Действие: %s\nРезультат:\n%s", intentType, actionResult)

	text, err := f.callLLM(ctx, formatterPrompt, userPrompt)
	if err != nil {
		// Formatting failed — return raw result, don't break the flow.
		slog.WarnContext(ctx, "Formatter.Format: LLM call failed, returning raw result", slog.String("error", err.Error()), slog.Duration("elapsed", time.Since(start)))
		return actionResult, nil //nolint:nilerr
	}

	slog.InfoContext(ctx, "Formatter.Format: done", slog.Int("formatted_len", len(text)), slog.Duration("elapsed", time.Since(start)))
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

func (f *Formatter) callLLM(ctx context.Context, sysPrompt, userMsg string) (string, error) {
	switch f.provider {
	case domain.ProviderClaude:
		return f.callClaude(ctx, sysPrompt, userMsg)
	case domain.ProviderOpenAI, domain.ProviderGrok:
		return f.callOpenAICompat(ctx, sysPrompt, userMsg)
	case domain.ProviderOllama:
		return f.callOllama(ctx, sysPrompt, userMsg)
	default:
		return "", domain.ErrUnsupportedProvider
	}
}

func (f *Formatter) callClaude(ctx context.Context, sysPrompt, userMsg string) (string, error) {
	slog.DebugContext(ctx, "Formatter.callClaude", slog.String("model", f.model))
	url := "https://api.anthropic.com/v1/messages"
	body := map[string]any{
		"model":      f.model,
		"max_tokens": 512,
		"system":     sysPrompt,
		"messages":   []map[string]string{{"role": "user", "content": userMsg}},
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("llm.Formatter.callClaude: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", f.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm.Formatter.callClaude: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("llm.Formatter.callClaude: %w", err)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("llm.Formatter.callClaude: empty response")
	}
	return result.Content[0].Text, nil
}

func (f *Formatter) callOpenAICompat(ctx context.Context, sysPrompt, userMsg string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	if f.provider == domain.ProviderGrok {
		url = "https://api.x.ai/v1/chat/completions"
	}
	slog.DebugContext(ctx, "Formatter.callOpenAICompat", slog.String("model", f.model), slog.String("provider", string(f.provider)))
	body := map[string]any{
		"model":      f.model,
		"max_tokens": 512,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": userMsg},
		},
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("llm.Formatter.callOpenAI: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.apiKey)

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm.Formatter.callOpenAI: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("llm.Formatter.callOpenAI: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("llm.Formatter.callOpenAI: empty response")
	}
	return result.Choices[0].Message.Content, nil
}

func (f *Formatter) callOllama(ctx context.Context, sysPrompt, userMsg string) (string, error) {
	slog.DebugContext(ctx, "Formatter.callOllama", slog.String("model", f.model), slog.String("base_url", f.baseURL))
	url := f.baseURL + "/api/chat"
	body := map[string]any{
		"model":  f.model,
		"stream": false,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": userMsg},
		},
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("llm.Formatter.callOllama: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm.Formatter.callOllama: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("llm.Formatter.callOllama: %w", err)
	}
	return result.Message.Content, nil
}
