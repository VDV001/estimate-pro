// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"context"
	"errors"
	"net/http"
)

const (
	whisperAPIPath        = "/v1/audio/transcriptions"
	defaultWhisperBaseURL = "https://api.openai.com"
	whisperDefaultModel   = "whisper-1"
)

// ErrWhisperNotImplemented marks the RED-step stub. Removed once
// RecognizeAudio is implemented in the GREEN commit.
var ErrWhisperNotImplemented = errors.New("WhisperAdapter: not implemented")

// WhisperAdapter calls the OpenAI /v1/audio/transcriptions endpoint
// with a multipart/form-data body carrying the raw audio bytes plus a
// model selector. Implements usecase.SpeechRecognizer — declared in
// bot/usecase/ports.go and wired into BotUsecase via the composition
// root. Telegram voice messages arrive as ogg/opus; Whisper accepts
// that format directly, so the adapter forwards the bytes verbatim.
type WhisperAdapter struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewWhisperAdapter constructs a Whisper adapter with the default 60s
// HTTP timeout. Empty model falls back to "whisper-1"; empty baseURL
// falls back to the OpenAI production endpoint so production wiring
// can pass "" for both.
func NewWhisperAdapter(apiKey, model, baseURL string) *WhisperAdapter {
	if baseURL == "" {
		baseURL = defaultWhisperBaseURL
	}
	if model == "" {
		model = whisperDefaultModel
	}
	return &WhisperAdapter{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: visionDefaultHTTPTimeout},
	}
}

// NewWhisperAdapterWithClient is the test seam — lets unit tests
// inject an httptest.Server-backed client.
func NewWhisperAdapterWithClient(apiKey, model, baseURL string, client *http.Client) *WhisperAdapter {
	if baseURL == "" {
		baseURL = defaultWhisperBaseURL
	}
	if model == "" {
		model = whisperDefaultModel
	}
	return &WhisperAdapter{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  client,
	}
}

// RecognizeAudio RED stub — see ErrWhisperNotImplemented. The
// GREEN-step commit replaces this with a real Whisper call.
func (a *WhisperAdapter) RecognizeAudio(ctx context.Context, audioBytes []byte, mimeType string) (string, error) {
	_ = ctx
	_ = audioBytes
	_ = mimeType
	return "", ErrWhisperNotImplemented
}
