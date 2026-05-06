// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/llm"
	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/usecase"
)

// fakeJPEG is a tiny payload — the adapter never decodes it, only
// base64-encodes and forwards. Real JPEGs would balloon the test for
// no benefit.
var fakeJPEG = []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}

// TestClaudeVisionAdapter_ImplementsTextExtractor pins the structural
// contract — if the port signature drifts, this fails to compile. No
// behavioural assertion needed.
func TestClaudeVisionAdapter_ImplementsTextExtractor(t *testing.T) {
	t.Parallel()
	var _ usecase.TextExtractor = (*llm.ClaudeVisionAdapter)(nil)
}

func TestClaudeVisionAdapter_ExtractTextFromImage_HappyPath(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var capturedBody []byte
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		capturedHeaders = r.Header.Clone()
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"text":"Привет, мир!"}],"usage":{"input_tokens":120,"output_tokens":4}}`))
	}))
	defer server.Close()

	adapter := llm.NewClaudeVisionAdapterWithClient("test-key", "claude-3-5-sonnet-20241022", server.URL, server.Client())

	got, err := adapter.ExtractTextFromImage(ctx, fakeJPEG)
	if err != nil {
		t.Fatalf("ExtractTextFromImage: unexpected error: %v", err)
	}
	if got != "Привет, мир!" {
		t.Fatalf("ExtractTextFromImage: got %q, want %q", got, "Привет, мир!")
	}

	if h := capturedHeaders.Get("x-api-key"); h != "test-key" {
		t.Errorf("x-api-key header: got %q, want %q", h, "test-key")
	}
	if h := capturedHeaders.Get("anthropic-version"); h == "" {
		t.Errorf("anthropic-version header: empty")
	}

	var payload struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content []struct {
				Type   string `json:"type"`
				Text   string `json:"text,omitempty"`
				Source *struct {
					Type      string `json:"type"`
					MediaType string `json:"media_type"`
					Data      string `json:"data"`
				} `json:"source,omitempty"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("decode request body: %v\nbody: %s", err, capturedBody)
	}
	if payload.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("model: got %q, want %q", payload.Model, "claude-3-5-sonnet-20241022")
	}
	if len(payload.Messages) != 1 || payload.Messages[0].Role != "user" {
		t.Fatalf("messages: got %+v", payload.Messages)
	}
	content := payload.Messages[0].Content
	if len(content) != 2 {
		t.Fatalf("content blocks: got %d, want 2 (image + text)", len(content))
	}

	var imgBlock, txtBlock int = -1, -1
	for i, b := range content {
		switch b.Type {
		case "image":
			imgBlock = i
		case "text":
			txtBlock = i
		}
	}
	if imgBlock < 0 || txtBlock < 0 {
		t.Fatalf("expected one image + one text block, got %+v", content)
	}
	src := content[imgBlock].Source
	if src == nil {
		t.Fatalf("image block: source is nil")
	}
	if src.Type != "base64" {
		t.Errorf("image source type: got %q, want %q", src.Type, "base64")
	}
	if !strings.HasPrefix(src.MediaType, "image/") {
		t.Errorf("image media type: got %q, want image/*", src.MediaType)
	}
	decoded, err := base64.StdEncoding.DecodeString(src.Data)
	if err != nil {
		t.Fatalf("decode base64 image: %v", err)
	}
	if string(decoded) != string(fakeJPEG) {
		t.Errorf("image bytes round-trip mismatch")
	}
	if content[txtBlock].Text == "" {
		t.Errorf("text block: empty prompt")
	}
}

func TestClaudeVisionAdapter_ExtractTextFromImage_HTTPError(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"upstream"}`))
	}))
	defer server.Close()

	adapter := llm.NewClaudeVisionAdapterWithClient("test-key", "claude-3-5-sonnet-20241022", server.URL, server.Client())
	if _, err := adapter.ExtractTextFromImage(ctx, fakeJPEG); err == nil {
		t.Fatal("ExtractTextFromImage: expected error on 500, got nil")
	}
}

func TestClaudeVisionAdapter_ExtractTextFromImage_EmptyContent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[]}`))
	}))
	defer server.Close()

	adapter := llm.NewClaudeVisionAdapterWithClient("test-key", "claude-3-5-sonnet-20241022", server.URL, server.Client())
	if _, err := adapter.ExtractTextFromImage(ctx, fakeJPEG); err == nil {
		t.Fatal("ExtractTextFromImage: expected error on empty content, got nil")
	}
}
