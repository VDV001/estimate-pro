// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

const (
	whisperAPIPath        = "/v1/audio/transcriptions"
	defaultWhisperBaseURL = "https://api.openai.com"
	whisperDefaultModel   = "whisper-1"
)

// ErrWhisperHTTP marks a non-2xx response from the OpenAI endpoint.
var ErrWhisperHTTP = errors.New("WhisperAdapter: HTTP error")

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

type whisperResponse struct {
	Text string `json:"text"`
}

// RecognizeAudio uploads audioBytes to the Whisper endpoint as
// multipart/form-data with fields {file, model} and returns the
// transcript. Whisper auto-detects the audio container from the file
// extension carried by the multipart filename, so the adapter maps
// mimeType to a sensible extension (ogg/opus → .ogg, mpeg → .mp3,
// wav → .wav, anything else → .ogg fallback since Telegram voice
// messages are always ogg/opus).
func (a *WhisperAdapter) RecognizeAudio(ctx context.Context, audioBytes []byte, mimeType string) (string, error) {
	start := time.Now()

	var bodyBuf bytes.Buffer
	mw := multipart.NewWriter(&bodyBuf)

	filename := whisperFilenameFor(mimeType)
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	if mimeType != "" {
		partHeader.Set("Content-Type", mimeType)
	}
	filePart, err := mw.CreatePart(partHeader)
	if err != nil {
		return "", fmt.Errorf("WhisperAdapter.RecognizeAudio: create file part: %w", err)
	}
	if _, err := filePart.Write(audioBytes); err != nil {
		return "", fmt.Errorf("WhisperAdapter.RecognizeAudio: write audio: %w", err)
	}
	if err := mw.WriteField("model", a.model); err != nil {
		return "", fmt.Errorf("WhisperAdapter.RecognizeAudio: write model: %w", err)
	}
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("WhisperAdapter.RecognizeAudio: close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+whisperAPIPath, &bodyBuf)
	if err != nil {
		return "", fmt.Errorf("WhisperAdapter.RecognizeAudio: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := a.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "WhisperAdapter.RecognizeAudio: HTTP error",
			slog.String("error", err.Error()),
			slog.Duration("elapsed", time.Since(start)))
		return "", fmt.Errorf("WhisperAdapter.RecognizeAudio: %w: %v", ErrWhisperHTTP, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("WhisperAdapter.RecognizeAudio: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "WhisperAdapter.RecognizeAudio: non-200",
			slog.Int("status", resp.StatusCode),
			slog.String("body_preview", visionBodyPreview(respBody)),
			slog.Duration("elapsed", time.Since(start)))
		return "", fmt.Errorf("WhisperAdapter.RecognizeAudio: %w: status %d", ErrWhisperHTTP, resp.StatusCode)
	}

	var wr whisperResponse
	if err := json.Unmarshal(respBody, &wr); err != nil {
		slog.ErrorContext(ctx, "WhisperAdapter.RecognizeAudio: unmarshal failed",
			slog.String("error", err.Error()),
			slog.String("body_preview", visionBodyPreview(respBody)))
		return "", fmt.Errorf("WhisperAdapter.RecognizeAudio: unmarshal: %w", err)
	}

	slog.InfoContext(ctx, "WhisperAdapter.RecognizeAudio: ok",
		slog.String("model", a.model),
		slog.Int("status", resp.StatusCode),
		slog.Int("text_len", len(wr.Text)),
		slog.Duration("elapsed", time.Since(start)))
	return wr.Text, nil
}

// whisperFilenameFor maps a Telegram voice mimeType to a filename
// Whisper can interpret. Telegram sends "audio/ogg" for voice
// messages; the fallback also picks .ogg because legitimate Telegram
// audio is always opus-in-ogg even when the upstream forgot the mime
// header.
func whisperFilenameFor(mimeType string) string {
	switch strings.ToLower(mimeType) {
	case "audio/mpeg", "audio/mp3":
		return "audio.mp3"
	case "audio/wav", "audio/x-wav":
		return "audio.wav"
	case "audio/mp4", "audio/m4a":
		return "audio.m4a"
	case "audio/webm":
		return "audio.webm"
	default:
		return "audio.ogg"
	}
}
