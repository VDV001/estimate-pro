package channel

import (
	"context"
	"fmt"

	"github.com/VDV001/estimate-pro/backend/internal/infra/composio"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
)

// EmailLookup resolves user ID to email address.
type EmailLookup interface {
	GetEmail(ctx context.Context, userID string) (string, error)
}

// EmailSender sends notifications via Gmail through Composio.
type EmailSender struct {
	client    *composio.Client
	accountID string
	lookup    EmailLookup
}

func NewEmailSender(client *composio.Client, accountID string, lookup EmailLookup) *EmailSender {
	return &EmailSender{client: client, accountID: accountID, lookup: lookup}
}

func (s *EmailSender) Channel() domain.Channel { return domain.ChannelEmail }

func (s *EmailSender) Send(ctx context.Context, userID, title, message string) error {
	email, err := s.lookup.GetEmail(ctx, userID)
	if err != nil {
		return fmt.Errorf("EmailSender.Send: lookup: %w", err)
	}

	params := map[string]any{
		"recipient_email": email,
		"subject":         title,
		"body":            message,
	}
	if err := s.client.ExecuteAction(ctx, "GMAIL_SEND_EMAIL", s.accountID, params); err != nil {
		return fmt.Errorf("EmailSender.Send: %w", err)
	}
	return nil
}
