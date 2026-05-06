// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"log/slog"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/handler/messages"
	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/telegram"
)

// recognizePhoto downloads the highest-resolution PhotoSize from the
// incoming Telegram message, runs OCR through TextExtractor, and
// returns the recognised text. ok=false means a soft failure has
// already been reported to the user (no extractor configured,
// download error, or OCR error) and ProcessMessage must stop.
//
// Telegram orders message.photo by ascending resolution, so the last
// element is always the largest available variant — picking it gives
// OCR the best raw signal at the cost of slightly more bytes
// downloaded (within Telegram's 20MB limit on photos).
func (uc *BotUsecase) recognizePhoto(ctx context.Context, msg *telegram.Message, chatID string) (string, bool) {
	if uc.textExtractor == nil {
		slog.WarnContext(ctx, "BotUsecase.recognizePhoto: textExtractor not configured", slog.String("chat_id", chatID))
		_ = uc.telegram.SendMessage(ctx, chatID, messages.PhotoRecognitionUnavailable)
		return "", false
	}
	if len(msg.Photo) == 0 {
		return "", false
	}

	largest := msg.Photo[len(msg.Photo)-1]
	slog.InfoContext(ctx, "BotUsecase.recognizePhoto: downloading photo",
		slog.String("file_id", largest.FileID),
		slog.Int("width", largest.Width),
		slog.Int("height", largest.Height),
		slog.Int64("file_size", largest.FileSize),
	)

	fileURL, err := uc.telegram.GetFileURL(ctx, largest.FileID)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.recognizePhoto: GetFileURL failed", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, messages.PhotoDownloadFailed)
		return "", false
	}
	data, err := uc.telegram.DownloadFile(ctx, fileURL)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.recognizePhoto: DownloadFile failed", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, messages.PhotoDownloadFailed)
		return "", false
	}

	_ = uc.telegram.SetReaction(ctx, chatID, msg.MessageID, "👀")

	text, err := uc.textExtractor.ExtractTextFromImage(ctx, data)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.recognizePhoto: ExtractTextFromImage failed", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, messages.PhotoRecognitionFailed)
		return "", false
	}

	slog.InfoContext(ctx, "BotUsecase.recognizePhoto: ok",
		slog.String("file_id", largest.FileID),
		slog.Int("text_len", len(text)),
	)
	return text, true
}

// recognizeVoice mirrors recognizePhoto for Telegram voice messages.
// Telegram always sends voice as ogg/opus; mimeType is forwarded to
// SpeechRecognizer so adapters can set a correct multipart filename
// (Whisper auto-detects format from the .ogg / .mp3 / .wav extension).
func (uc *BotUsecase) recognizeVoice(ctx context.Context, msg *telegram.Message, chatID string) (string, bool) {
	if uc.speechRecognizer == nil {
		slog.WarnContext(ctx, "BotUsecase.recognizeVoice: speechRecognizer not configured", slog.String("chat_id", chatID))
		_ = uc.telegram.SendMessage(ctx, chatID, messages.VoiceRecognitionUnavailable)
		return "", false
	}
	if msg.Voice == nil {
		return "", false
	}

	v := msg.Voice
	slog.InfoContext(ctx, "BotUsecase.recognizeVoice: downloading voice",
		slog.String("file_id", v.FileID),
		slog.String("mime_type", v.MimeType),
		slog.Int("duration", v.Duration),
		slog.Int64("file_size", v.FileSize),
	)

	fileURL, err := uc.telegram.GetFileURL(ctx, v.FileID)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.recognizeVoice: GetFileURL failed", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, messages.VoiceDownloadFailed)
		return "", false
	}
	data, err := uc.telegram.DownloadFile(ctx, fileURL)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.recognizeVoice: DownloadFile failed", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, messages.VoiceDownloadFailed)
		return "", false
	}

	_ = uc.telegram.SetReaction(ctx, chatID, msg.MessageID, "👀")

	text, err := uc.speechRecognizer.RecognizeAudio(ctx, data, v.MimeType)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.recognizeVoice: RecognizeAudio failed", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, messages.VoiceRecognitionFailed)
		return "", false
	}

	slog.InfoContext(ctx, "BotUsecase.recognizeVoice: ok",
		slog.String("file_id", v.FileID),
		slog.Int("text_len", len(text)),
	)
	return text, true
}
