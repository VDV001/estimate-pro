// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package usecase ports — interfaces declared on the consumer side of
// the dependency (Clean Architecture / DIP). Production implementations
// live in adapter packages (bot/llm, shared/llm) and are injected via
// the composition root in main.go. Tests use lightweight in-package
// fakes instead of mocks.
package usecase

import "context"

// TextExtractor recovers the textual content of a raster image. Used by
// the bot when a Telegram user attaches a photo: the recognised text is
// then fed back into the regular intent pipeline as if the user had
// typed it. Implementations call an LLM vision endpoint (Anthropic
// Claude Vision in production); the port deliberately knows nothing
// about model selection or media types — image bytes in, plain text
// out.
//
// Errors propagate without wrapping decisions baked into the port. The
// usecase layer maps them to a user-facing "не удалось распознать"
// message; downstream callers should not inspect the concrete error
// type.
type TextExtractor interface {
	ExtractTextFromImage(ctx context.Context, imageBytes []byte) (string, error)
}

// SpeechRecognizer transcribes spoken audio into text. Used by the bot
// when a Telegram user sends a voice message (.ogg / opus): the
// transcript is then handed to the intent pipeline as if it had been
// typed. Implementations call an STT endpoint (OpenAI Whisper in
// production). mimeType lets the adapter set the correct multipart
// content-type for providers that auto-detect format from it.
type SpeechRecognizer interface {
	RecognizeAudio(ctx context.Context, audioBytes []byte, mimeType string) (string, error)
}
