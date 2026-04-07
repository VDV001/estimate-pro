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

const defaultOpenAIBaseURL = "https://api.openai.com"

// OpenAIParser implements domain.LLMParser using the OpenAI Chat Completions API.
type OpenAIParser struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewOpenAIParser creates a new OpenAIParser with the given API key and model.
func NewOpenAIParser(apiKey, model string) *OpenAIParser {
	return &OpenAIParser{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultOpenAIBaseURL,
		client:  &http.Client{},
	}
}

// openaiRequest is the request body for the OpenAI Chat Completions API.
type openaiRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	Messages  []openaiMessage  `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiResponse is the response body from the OpenAI Chat Completions API.
type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ParseIntent parses a user message into a structured Intent using OpenAI.
func (p *OpenAIParser) ParseIntent(ctx context.Context, message string, history []string) (*domain.Intent, error) {
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
		return nil, fmt.Errorf("OpenAIParser.ParseIntent: marshal request: %w", err)
	}

	url := p.baseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("OpenAIParser.ParseIntent: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAIParser.ParseIntent: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("OpenAIParser.ParseIntent: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAIParser.ParseIntent: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var openaiResp openaiResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("OpenAIParser.ParseIntent: unmarshal response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAIParser.ParseIntent: no choices in response")
	}

	intent, err := parseIntentResponse([]byte(openaiResp.Choices[0].Message.Content))
	if err != nil {
		return nil, fmt.Errorf("OpenAIParser.ParseIntent: %w", err)
	}

	intent.RawText = message
	return intent, nil
}
