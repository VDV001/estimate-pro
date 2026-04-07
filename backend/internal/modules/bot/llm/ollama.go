// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

const defaultOllamaBaseURL = "http://localhost:11434"

// OllamaParser implements domain.LLMParser using the Ollama local API.
type OllamaParser struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaParser creates a new OllamaParser with the given base URL and model.
func NewOllamaParser(baseURL, model string) *OllamaParser {
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	return &OllamaParser{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// ollamaRequest is the request body for the Ollama Chat API.
type ollamaRequest struct {
	Model    string          `json:"model"`
	Stream   bool            `json:"stream"`
	Messages []ollamaMessage `json:"messages"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaResponse is the response body from the Ollama Chat API.
type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

// ParseIntent parses a user message into a structured Intent using Ollama.
func (p *OllamaParser) ParseIntent(ctx context.Context, message string, history []string) (*domain.Intent, error) {
	userPrompt := BuildUserPrompt(message, history)

	reqBody := ollamaRequest{
		Model:  p.model,
		Stream: false,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("OllamaParser.ParseIntent: marshal request: %w", err)
	}

	url := p.baseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("OllamaParser.ParseIntent: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OllamaParser.ParseIntent: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("OllamaParser.ParseIntent: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OllamaParser.ParseIntent: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("OllamaParser.ParseIntent: unmarshal response: %w", err)
	}

	intent, err := parseIntentResponse([]byte(ollamaResp.Message.Content))
	if err != nil {
		return nil, fmt.Errorf("OllamaParser.ParseIntent: %w", err)
	}

	intent.RawText = message
	return intent, nil
}
