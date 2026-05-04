// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	botllm "github.com/VDV001/estimate-pro/backend/internal/modules/bot/llm"
	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

// fakeSharedIntentParser captures the prompts the facade passes to the inner
// shared adapter and returns canned response/error.
type fakeSharedIntentParser struct {
	rawJSON     string
	usage       sharedllm.TokenUsage
	err         error
	gotSystem   string
	gotUser     string
	callCount   int
}

func (f *fakeSharedIntentParser) ParseIntent(ctx context.Context, systemPrompt, userPrompt string) (string, sharedllm.TokenUsage, error) {
	f.callCount++
	f.gotSystem = systemPrompt
	f.gotUser = userPrompt
	if f.err != nil {
		return "", sharedllm.ZeroTokenUsage(), f.err
	}
	return f.rawJSON, f.usage, nil
}

func TestBotIntentParser_BuildsPromptAndDelegatesToShared(t *testing.T) {
	t.Parallel()

	fake := &fakeSharedIntentParser{
		rawJSON: `{"type":"list_projects","params":{},"confidence":0.95}`,
		usage:   sharedllm.NewTokenUsage(120, 30),
	}
	facade := botllm.NewBotIntentParser(fake)

	intent, err := facade.ParseIntent(context.Background(), "покажи проекты", []string{"hello", "hi"})
	if err != nil {
		t.Fatalf("ParseIntent err = %v, want nil", err)
	}
	if fake.callCount != 1 {
		t.Errorf("inner.ParseIntent calls = %d, want 1", fake.callCount)
	}
	if fake.gotSystem == "" {
		t.Error("inner.ParseIntent received empty systemPrompt — facade must pass bot's systemPrompt")
	}
	expectedUser := botllm.BuildUserPrompt("покажи проекты", []string{"hello", "hi"})
	if fake.gotUser != expectedUser {
		t.Errorf("inner.ParseIntent userPrompt mismatch:\ngot:  %q\nwant: %q", fake.gotUser, expectedUser)
	}
	if intent == nil {
		t.Fatal("intent is nil, want unmarshalled domain.Intent")
	}
	if string(intent.Type) != "list_projects" {
		t.Errorf("intent.Type = %q, want %q", intent.Type, "list_projects")
	}
	if intent.Confidence != 0.95 {
		t.Errorf("intent.Confidence = %v, want 0.95", intent.Confidence)
	}
}

func TestBotIntentParser_AttachesRawText(t *testing.T) {
	t.Parallel()

	fake := &fakeSharedIntentParser{
		rawJSON: `{"type":"help","params":{},"confidence":0.9}`,
		usage:   sharedllm.ZeroTokenUsage(),
	}
	facade := botllm.NewBotIntentParser(fake)

	const message = "что ты умеешь?"
	intent, err := facade.ParseIntent(context.Background(), message, nil)
	if err != nil {
		t.Fatalf("ParseIntent err = %v, want nil", err)
	}
	if intent.RawText != message {
		t.Errorf("intent.RawText = %q, want %q", intent.RawText, message)
	}
}

func TestBotIntentParser_PropagatesSentinelErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		inner   error
		wantErr error
	}{
		{
			name:    "ErrLLMResponseInvalid",
			inner:   fmt.Errorf("decode: %w", sharedllm.ErrLLMResponseInvalid),
			wantErr: sharedllm.ErrLLMResponseInvalid,
		},
		{
			name:    "ErrLLMHTTP",
			inner:   fmt.Errorf("status 502: %w", sharedllm.ErrLLMHTTP),
			wantErr: sharedllm.ErrLLMHTTP,
		},
		{
			name:    "ErrLLMTimeout",
			inner:   fmt.Errorf("ctx deadline: %w", sharedllm.ErrLLMTimeout),
			wantErr: sharedllm.ErrLLMTimeout,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeSharedIntentParser{err: tc.inner}
			facade := botllm.NewBotIntentParser(fake)

			_, err := facade.ParseIntent(context.Background(), "msg", nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want errors.Is(%v)", err, tc.wantErr)
			}
		})
	}
}

func TestBotIntentParser_RejectsInvalidJSONFromInner(t *testing.T) {
	t.Parallel()

	fake := &fakeSharedIntentParser{
		rawJSON: "not a json",
		usage:   sharedllm.ZeroTokenUsage(),
	}
	facade := botllm.NewBotIntentParser(fake)

	_, err := facade.ParseIntent(context.Background(), "msg", nil)
	if err == nil {
		t.Fatal("expected JSON parse error, got nil")
	}
	if !strings.Contains(err.Error(), "BotIntentParser") {
		t.Errorf("err = %q, want wrap mentioning BotIntentParser", err.Error())
	}
}
