// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

func TestNewParser_AllProviders(t *testing.T) {
	tests := []struct {
		name     string
		provider domain.LLMProviderType
		wantType string
	}{
		{"claude", domain.ProviderClaude, "*llm.ClaudeParser"},
		{"openai", domain.ProviderOpenAI, "*llm.OpenAIParser"},
		{"grok", domain.ProviderGrok, "*llm.GrokParser"},
		{"ollama", domain.ProviderOllama, "*llm.OllamaParser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.provider, "test-key", "test-model", "http://localhost:11434")
			if err != nil {
				t.Fatalf("NewParser(%s) returned error: %v", tt.provider, err)
			}
			if parser == nil {
				t.Fatalf("NewParser(%s) returned nil parser", tt.provider)
			}
		})
	}
}

func TestNewParser_UnsupportedProvider(t *testing.T) {
	_, err := NewParser("unsupported", "key", "model", "")
	if err == nil {
		t.Fatal("NewParser(unsupported) should return error")
	}
	if !errors.Is(err, domain.ErrUnsupportedProvider) {
		t.Fatalf("expected ErrUnsupportedProvider, got: %v", err)
	}
}

func TestClaudeParser_ParseIntent(t *testing.T) {
	intentJSON := `{"type":"create_project","params":{"project_name":"Test Project"},"confidence":0.95}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("missing x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Error("missing anthropic-version header")
		}

		resp := claudeResponse{
			Content: []struct {
				Text string `json:"text"`
			}{
				{Text: intentJSON},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	parser := NewClaudeParser("test-key", "claude-3-haiku")
	parser.baseURL = srv.URL

	intent, err := parser.ParseIntent(t.Context(), "Create project Test Project", nil)
	if err != nil {
		t.Fatalf("ParseIntent returned error: %v", err)
	}

	assertIntent(t, intent, domain.IntentCreateProject, "Test Project", 0.95)
}

func TestOpenAIParser_ParseIntent(t *testing.T) {
	intentJSON := `{"type":"list_projects","params":{},"confidence":0.9}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing Authorization header")
		}

		resp := openaiResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: intentJSON}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	parser := NewOpenAIParser("test-key", "gpt-4")
	parser.baseURL = srv.URL

	intent, err := parser.ParseIntent(t.Context(), "Show my projects", nil)
	if err != nil {
		t.Fatalf("ParseIntent returned error: %v", err)
	}

	assertIntent(t, intent, domain.IntentListProjects, "", 0.9)
}

func TestGrokParser_ParseIntent(t *testing.T) {
	intentJSON := `{"type":"add_member","params":{"project_name":"App","email":"test@example.com","role":"editor"},"confidence":0.88}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer grok-key" {
			t.Error("missing Authorization header")
		}

		resp := openaiResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: intentJSON}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	parser := NewGrokParser("grok-key", "grok-3-mini")
	parser.baseURL = srv.URL

	intent, err := parser.ParseIntent(t.Context(), "Add test@example.com as editor to App", nil)
	if err != nil {
		t.Fatalf("ParseIntent returned error: %v", err)
	}

	assertIntent(t, intent, domain.IntentAddMember, "App", 0.88)
	if intent.Params["email"] != "test@example.com" {
		t.Errorf("expected email=test@example.com, got %s", intent.Params["email"])
	}
}

func TestOllamaParser_ParseIntent(t *testing.T) {
	intentJSON := `{"type":"help","params":{},"confidence":0.99}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaResponse{}
		resp.Message.Content = intentJSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	parser := NewOllamaParser(srv.URL, "llama3")

	intent, err := parser.ParseIntent(t.Context(), "help", nil)
	if err != nil {
		t.Fatalf("ParseIntent returned error: %v", err)
	}

	assertIntent(t, intent, domain.IntentHelp, "", 0.99)
}

func TestParseIntentResponse_WithMarkdownWrapper(t *testing.T) {
	input := "```json\n{\"type\":\"list_projects\",\"params\":{},\"confidence\":0.9}\n```"

	intent, err := parseIntentResponse([]byte(input))
	if err != nil {
		t.Fatalf("parseIntentResponse returned error: %v", err)
	}

	if intent.Type != domain.IntentListProjects {
		t.Errorf("expected type=%s, got %s", domain.IntentListProjects, intent.Type)
	}
	if intent.Confidence != 0.9 {
		t.Errorf("expected confidence=0.9, got %f", intent.Confidence)
	}
}

func TestParseIntentResponse_InvalidJSON(t *testing.T) {
	_, err := parseIntentResponse([]byte("this is not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBuildUserPrompt_WithoutHistory(t *testing.T) {
	result := BuildUserPrompt("hello", nil)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestBuildUserPrompt_WithHistory(t *testing.T) {
	result := BuildUserPrompt("current message", []string{"prev1", "prev2"})
	if result == "current message" {
		t.Error("expected history to be included in prompt")
	}
}

// assertIntent is a test helper that verifies common intent fields.
func assertIntent(t *testing.T, intent *domain.Intent, expectedType domain.IntentType, expectedProjectName string, expectedConfidence float64) {
	t.Helper()

	if intent.Type != expectedType {
		t.Errorf("expected type=%s, got %s", expectedType, intent.Type)
	}
	if expectedProjectName != "" {
		if intent.Params["project_name"] != expectedProjectName {
			t.Errorf("expected project_name=%s, got %s", expectedProjectName, intent.Params["project_name"])
		}
	}
	if intent.Confidence != expectedConfidence {
		t.Errorf("expected confidence=%f, got %f", expectedConfidence, intent.Confidence)
	}
}

func TestClaudeParser_ParseIntent_WithHistory(t *testing.T) {
	intentJSON := `{"type":"create_project","params":{"project_name":"Test"},"confidence":0.9}`

	var receivedBody claudeRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)

		resp := claudeResponse{
			Content: []struct {
				Text string `json:"text"`
			}{
				{Text: intentJSON},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	parser := NewClaudeParser("test-key", "claude-3-haiku")
	parser.baseURL = srv.URL

	_, err := parser.ParseIntent(context.Background(), "create Test", []string{"hello", "hi there"})
	if err != nil {
		t.Fatalf("ParseIntent returned error: %v", err)
	}

	if len(receivedBody.Messages) == 0 {
		t.Fatal("expected messages in request body")
	}
	userContent := receivedBody.Messages[0].Content
	if userContent == "create Test" {
		t.Error("expected history to be included in user prompt")
	}
}
