// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
	"github.com/google/uuid"
)

// NotificationUsecase orchestrates notification creation, delivery, and preference management.
type NotificationUsecase struct {
	notifRepo domain.NotificationRepository
	prefRepo  domain.PreferenceRepository
	logRepo   domain.DeliveryLogRepository
	members   domain.MemberLister
	senders   map[domain.Channel]domain.ExternalSender
}

// New creates a new NotificationUsecase.
func New(
	notifRepo domain.NotificationRepository,
	prefRepo domain.PreferenceRepository,
	logRepo domain.DeliveryLogRepository,
	members domain.MemberLister,
	senders ...domain.ExternalSender,
) *NotificationUsecase {
	sm := make(map[domain.Channel]domain.ExternalSender, len(senders))
	for _, s := range senders {
		sm[s.Channel()] = s
	}
	return &NotificationUsecase{
		notifRepo: notifRepo,
		prefRepo:  prefRepo,
		logRepo:   logRepo,
		members:   members,
		senders:   sm,
	}
}

// Dispatch creates in-app notifications for all project members (excluding actor)
// and sends external notifications based on user preferences.
func (uc *NotificationUsecase) Dispatch(ctx context.Context, event domain.NotifyEvent) error {
	userIDs, err := uc.members.ListMemberUserIDs(ctx, event.ProjectID)
	if err != nil {
		return fmt.Errorf("NotificationUsecase.Dispatch: %w", err)
	}

	// Filter out actor.
	var recipients []string
	for _, uid := range userIDs {
		if uid != event.ActorID {
			recipients = append(recipients, uid)
		}
	}

	if len(recipients) == 0 {
		return nil
	}

	// Create in-app notifications in batch.
	now := time.Now()
	notifications := make([]*domain.Notification, 0, len(recipients))
	for _, uid := range recipients {
		notifications = append(notifications, &domain.Notification{
			ID:        uuid.New().String(),
			UserID:    uid,
			EventType: event.EventType,
			Title:     event.Title,
			Message:   event.Message,
			ProjectID: event.ProjectID,
			Read:      false,
			CreatedAt: now,
		})
	}

	if err := uc.notifRepo.CreateBatch(ctx, notifications); err != nil {
		return fmt.Errorf("NotificationUsecase.Dispatch: %w", err)
	}

	// Send via external channels based on preferences.
	for _, uid := range recipients {
		prefs, err := uc.prefRepo.Get(ctx, uid)
		if err != nil {
			slog.Error("failed to get notification preferences", "user", uid, "error", err)
			continue
		}

		enabledChannels := make(map[domain.Channel]bool)
		for _, p := range prefs {
			enabledChannels[p.Channel] = p.Enabled
		}

		for ch, sender := range uc.senders {
			if !enabledChannels[ch] {
				continue
			}

			status := "sent"
			sendErr := sender.Send(ctx, uid, event.Title, event.Message)
			if sendErr != nil {
				slog.Error("external notification send failed", "channel", ch, "user", uid, "error", sendErr)
				status = "failed"
			}

			if logErr := uc.logRepo.Create(ctx, &domain.DeliveryLog{
				ID:        uuid.New().String(),
				UserID:    uid,
				EventType: event.EventType,
				Channel:   ch,
				SentAt:    time.Now(),
				Status:    status,
			}); logErr != nil {
				slog.Error("failed to create delivery log", "user", uid, "channel", ch, "error", logErr)
			}
		}
	}

	return nil
}

// List returns paginated notifications for a user.
func (uc *NotificationUsecase) List(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error) {
	return uc.notifRepo.ListByUser(ctx, userID, limit, offset)
}

// CountUnread returns the number of unread notifications for a user.
func (uc *NotificationUsecase) CountUnread(ctx context.Context, userID string) (int, error) {
	return uc.notifRepo.CountUnread(ctx, userID)
}

// MarkRead marks a single notification as read.
func (uc *NotificationUsecase) MarkRead(ctx context.Context, userID, notificationID string) error {
	return uc.notifRepo.MarkRead(ctx, userID, notificationID)
}

// MarkAllRead marks all notifications as read for a user.
func (uc *NotificationUsecase) MarkAllRead(ctx context.Context, userID string) error {
	return uc.notifRepo.MarkAllRead(ctx, userID)
}

// GetPreferences returns a user's notification preferences, filling in defaults
// for any channels not explicitly set (in_app: enabled, email: disabled, telegram: disabled).
func (uc *NotificationUsecase) GetPreferences(ctx context.Context, userID string) ([]*domain.Preference, error) {
	prefs, err := uc.prefRepo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("NotificationUsecase.GetPreferences: %w", err)
	}

	existing := make(map[domain.Channel]bool, len(prefs))
	for _, p := range prefs {
		existing[p.Channel] = true
	}

	defaults := map[domain.Channel]bool{
		domain.ChannelInApp:    true,
		domain.ChannelEmail:    false,
		domain.ChannelTelegram: false,
	}

	for ch, enabled := range defaults {
		if !existing[ch] {
			prefs = append(prefs, &domain.Preference{
				UserID:  userID,
				Channel: ch,
				Enabled: enabled,
			})
		}
	}

	return prefs, nil
}

// SetPreference updates a user's preference for a given channel.
func (uc *NotificationUsecase) SetPreference(ctx context.Context, userID string, channel domain.Channel, enabled bool) error {
	if !channel.IsValid() {
		return domain.ErrInvalidChannel
	}

	return uc.prefRepo.Upsert(ctx, &domain.Preference{
		UserID:  userID,
		Channel: channel,
		Enabled: enabled,
	})
}
