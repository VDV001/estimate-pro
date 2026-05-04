// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

func newOllamaServerAndAdapter(body string, status int) (*httptest.Server, *llm.OllamaAdapter) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	return srv, llm.NewOllamaAdapter("llama-test", srv.URL)
}

func TestOllamaAdapter_Complete_ReturnsTextAndUsage(t *testing.T) {
	srv, a := newOllamaServerAndAdapter(
		`{"message":{"content":"hello world"},"prompt_eval_count":11,"eval_count":5}`,
		200,
	)
	defer srv.Close()

	text, usage, err := a.Complete(context.Background(), "system", "user", llm.CompletionOptions{})
	if err != nil {
		t.Fatalf("Complete: unexpected error %v", err)
	}
	if text != "hello world" {
		t.Errorf("text = %q, want %q", text, "hello world")
	}
	if usage.Prompt != 11 || usage.Completion != 5 || usage.Total != 16 {
		t.Errorf("usage = %+v, want {Prompt:11 Completion:5 Total:16}", usage)
	}
}

func TestOllamaAdapter_Complete_PropagatesNon200ViaSentinel(t *testing.T) {
	srv, a := newOllamaServerAndAdapter(`{"error":"model not found"}`, 500)
	defer srv.Close()

	_, _, err := a.Complete(context.Background(), "s", "u", llm.CompletionOptions{})
	if !errors.Is(err, llm.ErrLLMHTTP) {
		t.Errorf("error = %v, want errors.Is(ErrLLMHTTP)", err)
	}
}

func TestOllamaAdapter_Complete_PropagatesInvalidJSONViaSentinel(t *testing.T) {
	srv, a := newOllamaServerAndAdapter(`{not json`, 200)
	defer srv.Close()

	_, _, err := a.Complete(context.Background(), "s", "u", llm.CompletionOptions{})
	if !errors.Is(err, llm.ErrLLMResponseInvalid) {
		t.Errorf("error = %v, want errors.Is(ErrLLMResponseInvalid)", err)
	}
}

func TestOllamaAdapter_ParseIntent_DelegatesToComplete(t *testing.T) {
	srv, a := newOllamaServerAndAdapter(
		`{"message":{"content":"{\"type\":\"list_projects\"}"},"prompt_eval_count":3,"eval_count":4}`,
		200,
	)
	defer srv.Close()

	raw, usage, err := a.ParseIntent(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("ParseIntent: %v", err)
	}
	if raw != `{"type":"list_projects"}` {
		t.Errorf("raw = %q, want JSON intent", raw)
	}
	if usage.Total != 7 {
		t.Errorf("usage.Total = %d, want 7", usage.Total)
	}
}

func TestOllamaAdapter_DefaultBaseURL(t *testing.T) {
	a := llm.NewOllamaAdapter("llama-test", "")
	if a == nil {
		t.Fatal("NewOllamaAdapter returned nil for empty baseURL")
	}
}

func TestOllamaAdapter_SatisfiesInterfaces(t *testing.T) {
	var _ llm.Completer = (*llm.OllamaAdapter)(nil)
	var _ llm.IntentParser = (*llm.OllamaAdapter)(nil)
}
