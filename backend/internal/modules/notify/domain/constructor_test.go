// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
)

func TestNewNotification_Valid(t *testing.T) {
	n, err := domain.NewNotification("user-1", domain.EventMemberAdded, "You were added", "Alice added you to X", "proj-1")
	if err != nil {
		t.Fatalf("NewNotification: %v", err)
	}
	if n.ID == "" {
		t.Error("ID must be auto-generated")
	}
	if n.UserID != "user-1" || n.EventType != domain.EventMemberAdded {
		t.Errorf("fields wrong: %+v", n)
	}
	if n.Title != "You were added" || n.Message != "Alice added you to X" {
		t.Errorf("title/message wrong: %+v", n)
	}
	if n.ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q", n.ProjectID)
	}
	if n.Read {
		t.Error("Read must default to false")
	}
	if n.CreatedAt.IsZero() {
		t.Error("CreatedAt must be set")
	}
}

func TestNewNotification_Validation(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		eventT  domain.EventType
		title   string
		want    error
	}{
		{"empty user", "", domain.EventMemberAdded, "t", domain.ErrMissingUser},
		{"invalid event", "u1", domain.EventType("bogus"), "t", domain.ErrInvalidEventType},
		{"empty event", "u1", domain.EventType(""), "t", domain.ErrInvalidEventType},
		{"empty title", "u1", domain.EventMemberAdded, "", domain.ErrMissingTitle},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewNotification(tc.userID, tc.eventT, tc.title, "msg", "")
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNotification_MarkRead(t *testing.T) {
	n, _ := domain.NewNotification("u1", domain.EventMemberAdded, "t", "m", "")
	n.MarkRead()
	if !n.Read {
		t.Error("Read must be true after MarkRead")
	}
}

func TestNewPreference_Valid(t *testing.T) {
	p, err := domain.NewPreference("user-1", domain.ChannelEmail, true)
	if err != nil {
		t.Fatalf("NewPreference: %v", err)
	}
	if p.UserID != "user-1" || p.Channel != domain.ChannelEmail || !p.Enabled {
		t.Errorf("fields wrong: %+v", p)
	}
}

func TestNewPreference_Validation(t *testing.T) {
	_, err := domain.NewPreference("", domain.ChannelEmail, true)
	if !errors.Is(err, domain.ErrMissingUser) {
		t.Errorf("empty user: err = %v, want ErrMissingUser", err)
	}
	_, err = domain.NewPreference("u1", domain.Channel("bogus"), true)
	if !errors.Is(err, domain.ErrInvalidChannel) {
		t.Errorf("invalid channel: err = %v, want ErrInvalidChannel", err)
	}
}

func TestNewDeliveryLog_Valid(t *testing.T) {
	dl, err := domain.NewDeliveryLog("user-1", domain.EventMemberAdded, domain.ChannelEmail, "sent")
	if err != nil {
		t.Fatalf("NewDeliveryLog: %v", err)
	}
	if dl.ID == "" {
		t.Error("ID must be auto-generated")
	}
	if dl.UserID != "user-1" || dl.Channel != domain.ChannelEmail || dl.Status != "sent" {
		t.Errorf("fields wrong: %+v", dl)
	}
	if dl.SentAt.IsZero() {
		t.Error("SentAt must be set")
	}
}

func TestNewDeliveryLog_Validation(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		eventT  domain.EventType
		channel domain.Channel
		want    error
	}{
		{"empty user", "", domain.EventMemberAdded, domain.ChannelEmail, domain.ErrMissingUser},
		{"invalid event", "u1", domain.EventType("bogus"), domain.ChannelEmail, domain.ErrInvalidEventType},
		{"invalid channel", "u1", domain.EventMemberAdded, domain.Channel("bogus"), domain.ErrInvalidChannel},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewDeliveryLog(tc.userID, tc.eventT, tc.channel, "sent")
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}
