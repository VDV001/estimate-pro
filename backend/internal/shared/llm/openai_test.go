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

func newOpenAIAdapterWithCanned(body string, status int) *llm.OpenAIAdapter {
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})
	return llm.NewOpenAIAdapterWithClient(
		"test-key",
		"gpt-test",
		"https://api.openai.com",
		&http.Client{Transport: rt},
	)
}

func TestOpenAIAdapter_Complete_ReturnsTextAndUsage(t *testing.T) {
	a := newOpenAIAdapterWithCanned(
		`{"choices":[{"message":{"content":"hello world"}}],"usage":{"prompt_tokens":7,"completion_tokens":3}}`,
		200,
	)
	text, usage, err := a.Complete(context.Background(), "system", "user", llm.CompletionOptions{})
	if err != nil {
		t.Fatalf("Complete: unexpected error %v", err)
	}
	if text != "hello world" {
		t.Errorf("text = %q, want %q", text, "hello world")
	}
	if usage.Prompt != 7 || usage.Completion != 3 || usage.Total != 10 {
		t.Errorf("usage = %+v, want {Prompt:7 Completion:3 Total:10}", usage)
	}
}

func TestOpenAIAdapter_Complete_PropagatesNon200ViaSentinel(t *testing.T) {
	a := newOpenAIAdapterWithCanned(`{"error":{"message":"rate limit"}}`, 429)
	_, _, err := a.Complete(context.Background(), "s", "u", llm.CompletionOptions{})
	if !errors.Is(err, llm.ErrLLMHTTP) {
		t.Errorf("error = %v, want errors.Is(ErrLLMHTTP)", err)
	}
}

func TestOpenAIAdapter_Complete_PropagatesInvalidJSONViaSentinel(t *testing.T) {
	a := newOpenAIAdapterWithCanned(`{not json`, 200)
	_, _, err := a.Complete(context.Background(), "s", "u", llm.CompletionOptions{})
	if !errors.Is(err, llm.ErrLLMResponseInvalid) {
		t.Errorf("error = %v, want errors.Is(ErrLLMResponseInvalid)", err)
	}
}

func TestOpenAIAdapter_Complete_NoChoicesViaSentinel(t *testing.T) {
	a := newOpenAIAdapterWithCanned(`{"choices":[],"usage":{"prompt_tokens":1,"completion_tokens":0}}`, 200)
	_, _, err := a.Complete(context.Background(), "s", "u", llm.CompletionOptions{})
	if !errors.Is(err, llm.ErrLLMResponseInvalid) {
		t.Errorf("error = %v, want errors.Is(ErrLLMResponseInvalid)", err)
	}
}

func TestOpenAIAdapter_ParseIntent_DelegatesToComplete(t *testing.T) {
	a := newOpenAIAdapterWithCanned(
		`{"choices":[{"message":{"content":"{\"type\":\"list_projects\"}"}}],"usage":{"prompt_tokens":4,"completion_tokens":2}}`,
		200,
	)
	raw, usage, err := a.ParseIntent(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("ParseIntent: %v", err)
	}
	if raw != `{"type":"list_projects"}` {
		t.Errorf("raw = %q, want JSON intent", raw)
	}
	if usage.Total != 6 {
		t.Errorf("usage.Total = %d, want 6", usage.Total)
	}
}

func TestOpenAIAdapter_SatisfiesInterfaces(t *testing.T) {
	var _ llm.Completer = (*llm.OpenAIAdapter)(nil)
	var _ llm.IntentParser = (*llm.OpenAIAdapter)(nil)
}
