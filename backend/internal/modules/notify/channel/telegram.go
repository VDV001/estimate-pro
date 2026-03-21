package channel

import (
	"context"
	"fmt"

	"github.com/VDV001/estimate-pro/backend/internal/infra/composio"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
)

// TelegramChatLookup resolves user ID to Telegram chat ID.
type TelegramChatLookup interface {
	GetTelegramChatID(ctx context.Context, userID string) (string, error)
}

// TelegramSender sends notifications via Telegram through Composio.
type TelegramSender struct {
	client    *composio.Client
	accountID string
	lookup    TelegramChatLookup
}

func NewTelegramSender(client *composio.Client, accountID string, lookup TelegramChatLookup) *TelegramSender {
	return &TelegramSender{client: client, accountID: accountID, lookup: lookup}
}

func (s *TelegramSender) Channel() domain.Channel { return domain.ChannelTelegram }

func (s *TelegramSender) Send(ctx context.Context, userID, title, message string) error {
	chatID, err := s.lookup.GetTelegramChatID(ctx, userID)
	if err != nil {
		return fmt.Errorf("TelegramSender.Send: lookup: %w", err)
	}

	text := fmt.Sprintf("*%s*\n%s", title, message)
	params := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	if err := s.client.ExecuteAction(ctx, "TELEGRAM_BOT_SEND_MESSAGE", s.accountID, params); err != nil {
		return fmt.Errorf("TelegramSender.Send: %w", err)
	}
	return nil
}
