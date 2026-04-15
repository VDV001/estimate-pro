package domain_test

import (
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
)

func TestChannel_IsValid(t *testing.T) {
	tests := []struct {
		channel domain.Channel
		want    bool
	}{
		{domain.ChannelInApp, true},
		{domain.ChannelEmail, true},
		{domain.ChannelTelegram, true},
		{"sms", false},
		{"", false},
		{"push", false},
	}

	for _, tc := range tests {
		t.Run(string(tc.channel), func(t *testing.T) {
			if got := tc.channel.IsValid(); got != tc.want {
				t.Errorf("Channel(%q).IsValid() = %v, want %v", tc.channel, got, tc.want)
			}
		})
	}
}

func TestNotifyEvent_Fields(t *testing.T) {
	evt := domain.NotifyEvent{
		EventType: domain.EventMemberAdded,
		ProjectID: "proj-1",
		ActorID:   "user-1",
		Title:     "Member added",
		Message:   "Alice was added",
	}

	if evt.EventType != domain.EventMemberAdded {
		t.Errorf("EventType: got %q, want %q", evt.EventType, domain.EventMemberAdded)
	}
	if evt.ProjectID != "proj-1" {
		t.Errorf("ProjectID: got %q", evt.ProjectID)
	}
}

func TestNotification_Struct(t *testing.T) {
	n := domain.Notification{
		ID:        "n-1",
		UserID:    "u-1",
		EventType: domain.EventDocumentUploaded,
		Title:     "New doc",
		Message:   "file.pdf uploaded",
		ProjectID: "p-1",
		Read:      false,
	}

	if n.Read {
		t.Error("expected notification to be unread")
	}
	if n.EventType != domain.EventDocumentUploaded {
		t.Errorf("EventType: got %q", n.EventType)
	}
}

func TestDeliveryLog_Struct(t *testing.T) {
	log := domain.DeliveryLog{
		ID:        "dl-1",
		UserID:    "u-1",
		EventType: domain.EventEstimationSubmitted,
		Channel:   domain.ChannelEmail,
		Status:    "sent",
	}

	if log.Status != "sent" {
		t.Errorf("Status: got %q", log.Status)
	}
}

func TestPreference_Struct(t *testing.T) {
	pref := domain.Preference{
		UserID:  "u-1",
		Channel: domain.ChannelTelegram,
		Enabled: true,
	}

	if !pref.Enabled {
		t.Error("expected preference to be enabled")
	}
}

func TestSentinelErrors(t *testing.T) {
	if domain.ErrNotificationNotFound == nil {
		t.Error("ErrNotificationNotFound should not be nil")
	}
	if domain.ErrInvalidChannel == nil {
		t.Error("ErrInvalidChannel should not be nil")
	}
	if domain.ErrDeliveryFailed == nil {
		t.Error("ErrDeliveryFailed should not be nil")
	}
}
