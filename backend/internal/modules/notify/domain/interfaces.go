// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "context"

// NotificationRepository handles in-app notification persistence.
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	CreateBatch(ctx context.Context, notifications []*Notification) error
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Notification, int, error)
	CountUnread(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) error
}

// PreferenceRepository handles notification channel preferences.
type PreferenceRepository interface {
	Get(ctx context.Context, userID string) ([]*Preference, error)
	Upsert(ctx context.Context, pref *Preference) error
}

// DeliveryLogRepository records external notification deliveries.
type DeliveryLogRepository interface {
	Create(ctx context.Context, log *DeliveryLog) error
}

// ExternalSender sends notifications via an external channel (email, telegram).
type ExternalSender interface {
	Channel() Channel
	Send(ctx context.Context, userID, title, message string) error
}

// MemberLister returns user IDs of project members (injected from project module).
type MemberLister interface {
	ListMemberUserIDs(ctx context.Context, projectID string) ([]string, error)
}
