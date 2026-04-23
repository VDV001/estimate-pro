// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client := NewClient("test-token")
	client.baseURL = srv.URL
	return client
}

func TestSendMessage_Success(t *testing.T) {
	var gotReq SendMessageRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sendMessage" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("unexpected Content-Type: %s", ct)
		}

		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	})

	err := client.SendMessage(t.Context(), "12345", "Hello, World!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotReq.ChatID != "12345" {
		t.Errorf("chat_id = %q, want %q", gotReq.ChatID, "12345")
	}
	if gotReq.Text != "Hello, World!" {
		t.Errorf("text = %q, want %q", gotReq.Text, "Hello, World!")
	}
	if gotReq.ParseMode != "" {
		t.Errorf("parse_mode = %q, want empty", gotReq.ParseMode)
	}
}

func TestSendMessage_APIError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	})

	err := client.SendMessage(t.Context(), "99999", "Hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := "telegram.Client.SendMessage: API error: Bad Request: chat not found"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestSendMarkdown_Success(t *testing.T) {
	var gotReq SendMessageRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotReq)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	})

	err := client.SendMarkdown(t.Context(), "12345", "*bold* text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotReq.ParseMode != "Markdown" {
		t.Errorf("parse_mode = %q, want %q", gotReq.ParseMode, "Markdown")
	}
	if gotReq.Text != "*bold* text" {
		t.Errorf("text = %q, want %q", gotReq.Text, "*bold* text")
	}
}

func TestSendInlineKeyboard_Success(t *testing.T) {
	var gotReq SendMessageRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotReq)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	})

	keyboard := [][]domain.InlineKeyboardButton{
		{
			{Text: "Option A", CallbackData: "a"},
			{Text: "Option B", CallbackData: "b"},
		},
	}

	err := client.SendInlineKeyboard(t.Context(), "12345", "Choose:", keyboard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotReq.ReplyMarkup == nil {
		t.Fatal("reply_markup is nil")
	}
	if len(gotReq.ReplyMarkup.InlineKeyboard) != 1 {
		t.Fatalf("inline_keyboard rows = %d, want 1", len(gotReq.ReplyMarkup.InlineKeyboard))
	}
	if len(gotReq.ReplyMarkup.InlineKeyboard[0]) != 2 {
		t.Fatalf("inline_keyboard cols = %d, want 2", len(gotReq.ReplyMarkup.InlineKeyboard[0]))
	}
	if gotReq.ReplyMarkup.InlineKeyboard[0][0].Text != "Option A" {
		t.Errorf("button text = %q, want %q", gotReq.ReplyMarkup.InlineKeyboard[0][0].Text, "Option A")
	}
	if gotReq.ReplyMarkup.InlineKeyboard[0][1].CallbackData != "b" {
		t.Errorf("button callback_data = %q, want %q", gotReq.ReplyMarkup.InlineKeyboard[0][1].CallbackData, "b")
	}
}

func TestAnswerCallbackQuery_Success(t *testing.T) {
	var gotPayload struct {
		CallbackQueryID string `json:"callback_query_id"`
		Text            string `json:"text"`
	}

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/answerCallbackQuery" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotPayload)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	})

	err := client.AnswerCallbackQuery(t.Context(), "cb-123", "Done!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPayload.CallbackQueryID != "cb-123" {
		t.Errorf("callback_query_id = %q, want %q", gotPayload.CallbackQueryID, "cb-123")
	}
	if gotPayload.Text != "Done!" {
		t.Errorf("text = %q, want %q", gotPayload.Text, "Done!")
	}
}

func TestSetReaction_Success(t *testing.T) {
	var gotPayload struct {
		ChatID    string `json:"chat_id"`
		MessageID int64  `json:"message_id"`
		Reaction  []struct {
			Type  string `json:"type"`
			Emoji string `json:"emoji"`
		} `json:"reaction"`
	}

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/setMessageReaction" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotPayload)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	})

	err := client.SetReaction(t.Context(), "12345", 42, "🚀")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPayload.ChatID != "12345" {
		t.Errorf("chat_id = %q, want %q", gotPayload.ChatID, "12345")
	}
	if gotPayload.MessageID != 42 {
		t.Errorf("message_id = %d, want 42", gotPayload.MessageID)
	}
	if len(gotPayload.Reaction) != 1 {
		t.Fatalf("reaction count = %d, want 1", len(gotPayload.Reaction))
	}
	if gotPayload.Reaction[0].Emoji != "🚀" {
		t.Errorf("emoji = %q, want 🚀", gotPayload.Reaction[0].Emoji)
	}
	if gotPayload.Reaction[0].Type != "emoji" {
		t.Errorf("type = %q, want emoji", gotPayload.Reaction[0].Type)
	}
}

func TestSetReaction_APIError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: message not found"}`))
	})

	err := client.SetReaction(t.Context(), "12345", 999, "🔥")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDownloadFile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("file content here"))
	}))
	t.Cleanup(srv.Close)

	client := NewClient("test-token")
	data, err := client.DownloadFile(t.Context(), srv.URL+"/file/test.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data) != "file content here" {
		t.Errorf("data = %q, want %q", string(data), "file content here")
	}
}

func TestDownloadFile_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	client := NewClient("test-token")
	_, err := client.DownloadFile(t.Context(), srv.URL+"/file/missing.pdf")
	if err == nil {
		t.Fatal("expected error for 404 status, got nil")
	}
}

func TestGetFileURL_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getFile" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var payload struct {
			FileID string `json:"file_id"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &payload)

		if payload.FileID != "file-abc" {
			t.Errorf("file_id = %q, want %q", payload.FileID, "file-abc")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":{"file_path":"documents/file_0.pdf"}}`))
	})

	url, err := client.GetFileURL(t.Context(), "file-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "https://api.telegram.org/file/bottest-token/documents/file_0.pdf"
	if url != want {
		t.Errorf("url = %q, want %q", url, want)
	}
}

// --- Issue #14: retry transient errors ---

func TestDoRequest_RetriesOnServerError(t *testing.T) {
	var attempts atomic.Int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"ok":false,"description":"Internal Server Error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	})

	err := client.SendMessage(t.Context(), "123", "hello")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestDoRequest_RetriesOnHTTP429(t *testing.T) {
	var attempts atomic.Int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"description":"Too Many Requests: retry after 1"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	})

	err := client.SendMessage(t.Context(), "123", "hello")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
}

func TestDoRequest_NoRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	})

	err := client.SendMessage(t.Context(), "123", "hello")
	if err == nil {
		t.Fatal("expected error on 4xx, got nil")
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on 4xx)", got)
	}
}

func TestDoRequest_GivesUpAfterMaxRetries(t *testing.T) {
	var attempts atomic.Int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Gateway"}`))
	})

	err := client.SendMessage(t.Context(), "123", "hello")
	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3 (initial + 2 retries)", got)
	}
}

// --- Issue #15: REACTION_INVALID should not be an error ---

func TestSetReaction_InvalidReactionSilenced(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: REACTION_INVALID"}`))
	})

	err := client.SetReaction(t.Context(), "12345", 42, "💡")
	if err != nil {
		t.Errorf("REACTION_INVALID should be silenced, got: %v", err)
	}
}
