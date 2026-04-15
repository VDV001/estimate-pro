package notify_test

import (
	"context"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/notify"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/usecase"
)

// --- Mock NotificationRepository ---

type mockNotifyRepo struct {
	created []*domain.Notification
}

func (m *mockNotifyRepo) Create(_ context.Context, n *domain.Notification) error {
	m.created = append(m.created, n)
	return nil
}

func (m *mockNotifyRepo) CreateBatch(_ context.Context, notifications []*domain.Notification) error {
	m.created = append(m.created, notifications...)
	return nil
}

func (m *mockNotifyRepo) ListByUser(_ context.Context, _ string, _, _ int) ([]*domain.Notification, int, error) {
	return nil, 0, nil
}

func (m *mockNotifyRepo) CountUnread(_ context.Context, _ string) (int, error) { return 0, nil }
func (m *mockNotifyRepo) MarkRead(_ context.Context, _, _ string) error        { return nil }
func (m *mockNotifyRepo) MarkAllRead(_ context.Context, _ string) error         { return nil }

// --- Mock PreferenceRepository ---

type mockPrefRepo struct{}

func (m *mockPrefRepo) Get(_ context.Context, _ string) ([]*domain.Preference, error) {
	return nil, nil
}
func (m *mockPrefRepo) Upsert(_ context.Context, _ *domain.Preference) error { return nil }

// --- Mock DeliveryLogRepository ---

type mockDeliveryRepo struct{}

func (m *mockDeliveryRepo) Create(_ context.Context, _ *domain.DeliveryLog) error { return nil }

// --- Mock MemberLister ---

type mockMemberLister struct {
	userIDs []string
}

func (m *mockMemberLister) ListMemberUserIDs(_ context.Context, _ string) ([]string, error) {
	return m.userIDs, nil
}

// --- Mock UserNameLookup ---

type mockNameLookup struct {
	names map[string]string
}

func (m *mockNameLookup) GetName(_ context.Context, userID string) (string, error) {
	if name, ok := m.names[userID]; ok {
		return name, nil
	}
	return userID, nil
}

func TestDispatcher_HandleEvent_KnownEvent(t *testing.T) {
	notifyRepo := &mockNotifyRepo{}
	uc := usecase.New(notifyRepo, &mockPrefRepo{}, &mockDeliveryRepo{}, &mockMemberLister{
		userIDs: []string{"u1", "u2"},
	})

	lookup := &mockNameLookup{names: map[string]string{"actor-1": "Alice"}}
	d := notify.NewDispatcher(uc, lookup, t.Context())

	d.HandleEvent("member.added", "proj-1", "actor-1")
	d.Shutdown()

	if len(notifyRepo.created) == 0 {
		t.Error("expected notifications to be created")
	}
}

func TestDispatcher_HandleEvent_UnknownEvent(t *testing.T) {
	notifyRepo := &mockNotifyRepo{}
	uc := usecase.New(notifyRepo, &mockPrefRepo{}, &mockDeliveryRepo{}, &mockMemberLister{
		userIDs: []string{"u1"},
	})

	lookup := &mockNameLookup{names: make(map[string]string)}
	d := notify.NewDispatcher(uc, lookup, t.Context())

	d.HandleEvent("unknown.event", "proj-1", "actor-1")
	d.Shutdown()

	if len(notifyRepo.created) != 0 {
		t.Error("expected no notifications for unknown event")
	}
}

func TestDispatcher_Shutdown_WaitsForPending(t *testing.T) {
	notifyRepo := &mockNotifyRepo{}
	uc := usecase.New(notifyRepo, &mockPrefRepo{}, &mockDeliveryRepo{}, &mockMemberLister{
		userIDs: []string{"u1", "u2", "u3"},
	})

	lookup := &mockNameLookup{names: make(map[string]string)}
	d := notify.NewDispatcher(uc, lookup, t.Context())

	// Fire multiple events.
	d.HandleEvent("document.uploaded", "proj-1", "actor-1")
	d.HandleEvent("estimation.submitted", "proj-2", "actor-2")

	done := make(chan struct{})
	go func() {
		d.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown did not complete in time")
	}
}

func TestDispatcher_AllEventTypes(t *testing.T) {
	events := []string{
		"member.added",
		"document.uploaded",
		"estimation.submitted",
		"estimation.aggregated",
	}

	for _, evt := range events {
		t.Run(evt, func(t *testing.T) {
			notifyRepo := &mockNotifyRepo{}
			uc := usecase.New(notifyRepo, &mockPrefRepo{}, &mockDeliveryRepo{}, &mockMemberLister{
				userIDs: []string{"u1"},
			})

			lookup := &mockNameLookup{names: map[string]string{"a": "Actor"}}
			d := notify.NewDispatcher(uc, lookup, t.Context())

			d.HandleEvent(evt, "proj-1", "a")
			d.Shutdown()

			if len(notifyRepo.created) == 0 {
				t.Errorf("expected notifications for event %q", evt)
			}
		})
	}
}
