package channel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/infra/composio"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
)

type mockEmailLookup struct {
	email string
	err   error
}

func (m *mockEmailLookup) GetEmail(_ context.Context, _ string) (string, error) {
	return m.email, m.err
}

type mockTelegramLookup struct {
	chatID string
	err    error
}

func (m *mockTelegramLookup) GetTelegramChatID(_ context.Context, _ string) (string, error) {
	return m.chatID, m.err
}

func TestEmailSender_Channel(t *testing.T) {
	s := &EmailSender{}
	if got := s.Channel(); got != domain.ChannelEmail {
		t.Errorf("Channel() = %v, want %v", got, domain.ChannelEmail)
	}
}

func TestEmailSender_Send(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		lookupErr  error
		statusCode int
		wantErr    bool
	}{
		{"success", "user@example.com", nil, 200, false},
		{"lookup_error", "", fmt.Errorf("not found"), 200, true},
		{"api_error", "user@example.com", nil, 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer srv.Close()

			client := composio.NewClient("key").WithBaseURL(srv.URL)
			sender := NewEmailSender(client, "acc-gmail", &mockEmailLookup{email: tt.email, err: tt.lookupErr})

			err := sender.Send(t.Context(), "user-1", "Test Subject", "Test Body")
			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTelegramSender_Channel(t *testing.T) {
	s := &TelegramSender{}
	if got := s.Channel(); got != domain.ChannelTelegram {
		t.Errorf("Channel() = %v, want %v", got, domain.ChannelTelegram)
	}
}

func TestTelegramSender_Send(t *testing.T) {
	tests := []struct {
		name       string
		chatID     string
		lookupErr  error
		statusCode int
		wantErr    bool
	}{
		{"success", "12345", nil, 200, false},
		{"lookup_error", "", fmt.Errorf("no chat id"), 200, true},
		{"api_error", "12345", nil, 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer srv.Close()

			client := composio.NewClient("key").WithBaseURL(srv.URL)
			sender := NewTelegramSender(client, "acc-tg", &mockTelegramLookup{chatID: tt.chatID, err: tt.lookupErr})

			err := sender.Send(t.Context(), "user-1", "Test Title", "Test Message")
			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
