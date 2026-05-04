// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

// roundTripFunc lets tests inject canned HTTP responses without spinning
// up an httptest.Server (Claude adapter has hardcoded api.anthropic.com URL).
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newClaudeAdapterWithCanned(body string, status int) *llm.ClaudeAdapter {
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})
	return llm.NewClaudeAdapterWithClient(
		"test-key",
		"claude-test",
		"https://api.anthropic.com",
		&http.Client{Transport: rt},
	)
}

func TestClaudeAdapter_Complete_ReturnsTextAndUsage(t *testing.T) {
	a := newClaudeAdapterWithCanned(
		`{"content":[{"text":"hello world"}],"usage":{"input_tokens":12,"output_tokens":4}}`,
		200,
	)
	text, usage, err := a.Complete(context.Background(), "system", "user", llm.CompletionOptions{})
	if err != nil {
		t.Fatalf("Complete: unexpected error %v", err)
	}
	if text != "hello world" {
		t.Errorf("text = %q, want %q", text, "hello world")
	}
	if usage.Prompt != 12 || usage.Completion != 4 || usage.Total != 16 {
		t.Errorf("usage = %+v, want {Prompt:12 Completion:4 Total:16}", usage)
	}
}

func TestClaudeAdapter_Complete_PropagatesNon200ViaSentinel(t *testing.T) {
	a := newClaudeAdapterWithCanned(`{"error":{"message":"unauthorized"}}`, 401)
	_, _, err := a.Complete(context.Background(), "s", "u", llm.CompletionOptions{})
	if !errors.Is(err, llm.ErrLLMHTTP) {
		t.Errorf("error = %v, want errors.Is(ErrLLMHTTP)", err)
	}
}

func TestClaudeAdapter_Complete_PropagatesInvalidJSONViaSentinel(t *testing.T) {
	a := newClaudeAdapterWithCanned(`{not json`, 200)
	_, _, err := a.Complete(context.Background(), "s", "u", llm.CompletionOptions{})
	if !errors.Is(err, llm.ErrLLMResponseInvalid) {
		t.Errorf("error = %v, want errors.Is(ErrLLMResponseInvalid)", err)
	}
}

func TestClaudeAdapter_Complete_EmptyContentViaSentinel(t *testing.T) {
	a := newClaudeAdapterWithCanned(`{"content":[],"usage":{"input_tokens":1,"output_tokens":0}}`, 200)
	_, _, err := a.Complete(context.Background(), "s", "u", llm.CompletionOptions{})
	if !errors.Is(err, llm.ErrLLMResponseInvalid) {
		t.Errorf("error = %v, want errors.Is(ErrLLMResponseInvalid)", err)
	}
}

func TestClaudeAdapter_ParseIntent_DelegatesToComplete(t *testing.T) {
	a := newClaudeAdapterWithCanned(
		`{"content":[{"text":"{\"type\":\"list_projects\"}"}],"usage":{"input_tokens":5,"output_tokens":3}}`,
		200,
	)
	raw, usage, err := a.ParseIntent(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("ParseIntent: %v", err)
	}
	if raw != `{"type":"list_projects"}` {
		t.Errorf("raw = %q, want JSON intent", raw)
	}
	if usage.Total != 8 {
		t.Errorf("usage.Total = %d, want 8", usage.Total)
	}
}

func TestClaudeAdapter_SatisfiesInterfaces(t *testing.T) {
	var _ llm.Completer = (*llm.ClaudeAdapter)(nil)
	var _ llm.IntentParser = (*llm.ClaudeAdapter)(nil)
}
