// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/handler"
	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/telegram"
	"github.com/go-chi/chi/v5"
)

type mockBotProcessor struct {
	processMessageFn  func(ctx context.Context, update *telegram.Update) error
	processCallbackFn func(ctx context.Context, update *telegram.Update) error
}

func (m *mockBotProcessor) ProcessMessage(ctx context.Context, update *telegram.Update) error {
	if m.processMessageFn != nil {
		return m.processMessageFn(ctx, update)
	}
	return nil
}

func (m *mockBotProcessor) ProcessCallback(ctx context.Context, update *telegram.Update) error {
	if m.processCallbackFn != nil {
		return m.processCallbackFn(ctx, update)
	}
	return nil
}

func TestHandleWebhook(t *testing.T) {
	const secret = "test-secret-token"

	tests := []struct {
		name               string
		secret             string
		body               string
		wantMessageCalled  bool
		wantCallbackCalled bool
	}{
		{
			name:   "ValidMessage",
			secret: secret,
			body: mustJSON(t, telegram.Update{
				UpdateID: 1,
				Message: &telegram.Message{
					MessageID: 10,
					Text:      "hello",
					Chat:      &telegram.Chat{ID: 123, Type: "private"},
				},
			}),
			wantMessageCalled: true,
		},
		{
			name:   "InvalidSecret",
			secret: "wrong-secret",
			body: mustJSON(t, telegram.Update{
				UpdateID: 2,
				Message: &telegram.Message{
					MessageID: 20,
					Text:      "hello",
					Chat:      &telegram.Chat{ID: 123, Type: "private"},
				},
			}),
			wantMessageCalled:  false,
			wantCallbackCalled: false,
		},
		{
			name:   "CallbackQuery",
			secret: secret,
			body: mustJSON(t, telegram.Update{
				UpdateID: 3,
				CallbackQuery: &telegram.CallbackQuery{
					ID:   "cb-1",
					From: &telegram.User{ID: 42, FirstName: "Test"},
					Data: "action:confirm",
				},
			}),
			wantCallbackCalled: true,
		},
		{
			name:   "InvalidJSON",
			secret: secret,
			body:   `{not valid json!!!`,
		},
		{
			name:   "EmptyUpdate",
			secret: secret,
			body:   mustJSON(t, telegram.Update{UpdateID: 5}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var messageCalled, callbackCalled bool

			mock := &mockBotProcessor{
				processMessageFn: func(_ context.Context, _ *telegram.Update) error {
					messageCalled = true
					return nil
				},
				processCallbackFn: func(_ context.Context, _ *telegram.Update) error {
					callbackCalled = true
					return nil
				},
			}

			h := handler.New(mock, secret)
			r := chi.NewRouter()
			h.Register(r)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/webhook", bytes.NewBufferString(tc.body))
			req.Header.Set("X-Telegram-Bot-Api-Secret-Token", tc.secret)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rec.Code)
			}

			if rec.Header().Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json, got %q", rec.Header().Get("Content-Type"))
			}

			if tc.wantMessageCalled != messageCalled {
				t.Errorf("processMessage called = %v, want %v", messageCalled, tc.wantMessageCalled)
			}

			if tc.wantCallbackCalled != callbackCalled {
				t.Errorf("processCallback called = %v, want %v", callbackCalled, tc.wantCallbackCalled)
			}
		})
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustJSON: %v", err)
	}
	return string(data)
}
