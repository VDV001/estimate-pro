// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
	botllm "github.com/VDV001/estimate-pro/backend/internal/modules/bot/llm"
	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

// fakeCompleter records the prompts/options the Formatter passes through
// and returns canned text/usage/err for assertion.
type fakeCompleter struct {
	text       string
	usage      sharedllm.TokenUsage
	err        error
	gotSystem  string
	gotUser    string
	gotOptions sharedllm.CompletionOptions
	calls      int
}

func (f *fakeCompleter) Complete(ctx context.Context, systemPrompt, userPrompt string, opts sharedllm.CompletionOptions) (string, sharedllm.TokenUsage, error) {
	f.calls++
	f.gotSystem = systemPrompt
	f.gotUser = userPrompt
	f.gotOptions = opts
	if f.err != nil {
		return "", sharedllm.ZeroTokenUsage(), f.err
	}
	return f.text, f.usage, nil
}

func TestFormatter_Format_DelegatesToCompleter(t *testing.T) {
	t.Parallel()

	c := &fakeCompleter{
		text:  "Готово, проект создан! 🚀",
		usage: sharedllm.NewTokenUsage(80, 30),
	}
	f := botllm.NewFormatter(c)

	out, err := f.Format(context.Background(), "raw action result", domain.IntentCreateProject)
	if err != nil {
		t.Fatalf("Format err = %v, want nil", err)
	}
	if out != "Готово, проект создан! 🚀" {
		t.Errorf("Format text = %q, want %q", out, "Готово, проект создан! 🚀")
	}
	if c.calls != 1 {
		t.Errorf("completer calls = %d, want 1", c.calls)
	}
	if c.gotSystem == "" {
		t.Error("Formatter must pass non-empty system prompt (Esti's personality) to completer")
	}
	if !strings.Contains(c.gotUser, "create_project") || !strings.Contains(c.gotUser, "raw action result") {
		t.Errorf("user prompt %q must contain intent + raw action result", c.gotUser)
	}
	if c.gotOptions.MaxTokens == 0 {
		t.Error("Formatter must set non-zero MaxTokens to bound formatter output length")
	}
}

func TestFormatter_Format_FallsBackOnCompleterError(t *testing.T) {
	t.Parallel()

	c := &fakeCompleter{err: errors.New("provider down")}
	f := botllm.NewFormatter(c)

	out, err := f.Format(context.Background(), "raw fallback text", domain.IntentHelp)
	if err != nil {
		t.Fatalf("Format must not propagate completer error, got %v", err)
	}
	if out != "raw fallback text" {
		t.Errorf("Format text = %q, want raw fallback %q", out, "raw fallback text")
	}
}

func TestFormatter_Format_FallsBackOnNilCompleter(t *testing.T) {
	t.Parallel()

	f := botllm.NewFormatter(nil)
	out, err := f.Format(context.Background(), "raw text without LLM", domain.IntentListProjects)
	if err != nil {
		t.Fatalf("Format err = %v, want nil", err)
	}
	if out != "raw text without LLM" {
		t.Errorf("Format text = %q, want raw fallback", out)
	}
}

func TestFormatter_Format_PreservesSentinelInWarn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{"http_failure", sharedllm.ErrLLMHTTP},
		{"invalid_response", sharedllm.ErrLLMResponseInvalid},
		{"timeout", sharedllm.ErrLLMTimeout},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &fakeCompleter{err: tc.err}
			f := botllm.NewFormatter(c)
			out, err := f.Format(context.Background(), "raw", domain.IntentHelp)
			if err != nil {
				t.Fatalf("Format must swallow %v error and return raw, got err=%v", tc.err, err)
			}
			if out != "raw" {
				t.Errorf("Format text = %q, want raw fallback", out)
			}
		})
	}
}

func TestFormatReaction(t *testing.T) {
	tests := []struct {
		intent domain.IntentType
		want   string
	}{
		{domain.IntentCreateProject, "🚀"},
		{domain.IntentAddMember, "🎉"},
		{domain.IntentSubmitEstimation, "🔥"},
		{domain.IntentUploadDocument, "👀"},
		{domain.IntentGetAggregated, "📊"},
		{domain.IntentListProjects, "👍"},
		{domain.IntentListMembers, "👍"},
		{domain.IntentGetProjectStatus, "👍"},
		{domain.IntentForgotPassword, "🔑"},
		{domain.IntentHelp, "💡"},
		{domain.IntentUnknown, ""},
		{"nonexistent_intent", ""},
	}

	for _, tc := range tests {
		t.Run(string(tc.intent), func(t *testing.T) {
			got := botllm.FormatReaction(tc.intent)
			if got != tc.want {
				t.Errorf("FormatReaction(%s) = %q, want %q", tc.intent, got, tc.want)
			}
		})
	}
}

func TestMentionReaction(t *testing.T) {
	if got := botllm.MentionReaction(); got != "👋" {
		t.Errorf("MentionReaction() = %q, want %q", got, "👋")
	}
}

func TestResponsePool_Pick(t *testing.T) {
	pools := []botllm.ResponsePool{
		botllm.UnlinkedUser,
		botllm.LowConfidence,
		botllm.LLMError,
		botllm.LLMConfigError,
		botllm.ExecuteError,
		botllm.Greeting,
		botllm.Thanks,
		botllm.SessionExpired,
		botllm.CancelConfirm,
		botllm.SuccessReaction,
	}

	for i, pool := range pools {
		if len(pool) == 0 {
			t.Errorf("pool[%d] is empty", i)
			continue
		}
		if pick := pool.Pick(); pick == "" {
			t.Errorf("pool[%d].Pick() returned empty string", i)
		}
	}
}
