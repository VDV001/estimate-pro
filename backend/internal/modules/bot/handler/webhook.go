// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/telegram"
	"github.com/go-chi/chi/v5"
)

// BotProcessor is the interface the handler needs from usecase.
type BotProcessor interface {
	ProcessMessage(ctx context.Context, update *telegram.Update) error
	ProcessCallback(ctx context.Context, update *telegram.Update) error
}

// Handler handles incoming Telegram webhook requests.
type Handler struct {
	botUC         BotProcessor
	webhookSecret string
}

// New creates a new webhook Handler.
func New(botUC BotProcessor, webhookSecret string) *Handler {
	return &Handler{
		botUC:         botUC,
		webhookSecret: webhookSecret,
	}
}

// Register mounts the webhook route on the given chi router.
func (h *Handler) Register(r chi.Router) {
	r.Post("/api/v1/bot/webhook", h.HandleWebhook)
}

// HandleWebhook processes incoming Telegram updates.
// It always returns 200 OK as required by the Telegram Bot API.
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validate webhook secret token.
	if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != h.webhookSecret {
		slog.WarnContext(ctx, "Handler.HandleWebhook: invalid webhook secret")
		writeOK(w)
		return
	}

	var update telegram.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		slog.ErrorContext(ctx, "Handler.HandleWebhook: failed to decode update", slog.String("error", err.Error()))
		writeOK(w)
		return
	}

	switch {
	case update.CallbackQuery != nil:
		if err := h.botUC.ProcessCallback(ctx, &update); err != nil {
			slog.ErrorContext(ctx, "Handler.HandleWebhook: ProcessCallback failed",
				slog.Int64("update_id", update.UpdateID),
				slog.String("error", err.Error()),
			)
		}
	case update.Message != nil:
		if err := h.botUC.ProcessMessage(ctx, &update); err != nil {
			slog.ErrorContext(ctx, "Handler.HandleWebhook: ProcessMessage failed",
				slog.Int64("update_id", update.UpdateID),
				slog.String("error", err.Error()),
			)
		}
	default:
		slog.DebugContext(ctx, "Handler.HandleWebhook: update has no message or callback query",
			slog.Int64("update_id", update.UpdateID),
		)
	}

	writeOK(w)
}

func writeOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{}`))
}
