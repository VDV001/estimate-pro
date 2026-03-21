package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/usecase"
)

// --- Mock implementations ---

type mockNotifRepo struct {
	createBatchFn func(ctx context.Context, n []*domain.Notification) error
	listFn        func(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error)
	countUnreadFn func(ctx context.Context, userID string) (int, error)
	markReadFn    func(ctx context.Context, userID, id string) error
	markAllReadFn func(ctx context.Context, userID string) error
}

func (m *mockNotifRepo) Create(_ context.Context, _ *domain.Notification) error { return nil }

func (m *mockNotifRepo) CreateBatch(ctx context.Context, n []*domain.Notification) error {
	if m.createBatchFn != nil {
		return m.createBatchFn(ctx, n)
	}
	return nil
}

func (m *mockNotifRepo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockNotifRepo) CountUnread(ctx context.Context, userID string) (int, error) {
	if m.countUnreadFn != nil {
		return m.countUnreadFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockNotifRepo) MarkRead(ctx context.Context, userID, id string) error {
	if m.markReadFn != nil {
		return m.markReadFn(ctx, userID, id)
	}
	return nil
}

func (m *mockNotifRepo) MarkAllRead(ctx context.Context, userID string) error {
	if m.markAllReadFn != nil {
		return m.markAllReadFn(ctx, userID)
	}
	return nil
}

type mockPrefRepo struct {
	getFn    func(ctx context.Context, userID string) ([]*domain.Preference, error)
	upsertFn func(ctx context.Context, pref *domain.Preference) error
}

func (m *mockPrefRepo) Get(ctx context.Context, userID string) ([]*domain.Preference, error) {
	if m.getFn != nil {
		return m.getFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockPrefRepo) Upsert(ctx context.Context, pref *domain.Preference) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, pref)
	}
	return nil
}

type mockLogRepo struct {
	createFn func(ctx context.Context, log *domain.DeliveryLog) error
}

func (m *mockLogRepo) Create(ctx context.Context, log *domain.DeliveryLog) error {
	if m.createFn != nil {
		return m.createFn(ctx, log)
	}
	return nil
}

type mockMembers struct {
	listFn func(ctx context.Context, projectID string) ([]string, error)
}

func (m *mockMembers) ListMemberUserIDs(ctx context.Context, projectID string) ([]string, error) {
	if m.listFn != nil {
		return m.listFn(ctx, projectID)
	}
	return nil, nil
}

type mockSender struct {
	channel domain.Channel
	sendFn  func(ctx context.Context, userID, title, message string) error
}

func (m *mockSender) Channel() domain.Channel { return m.channel }

func (m *mockSender) Send(ctx context.Context, userID, title, message string) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, userID, title, message)
	}
	return nil
}

// --- Tests ---

func TestDispatch_Success(t *testing.T) {
	var batchCreated []*domain.Notification
	var emailSentTo []string
	var logEntries []*domain.DeliveryLog

	notifRepo := &mockNotifRepo{
		createBatchFn: func(_ context.Context, n []*domain.Notification) error {
			batchCreated = n
			return nil
		},
	}

	prefRepo := &mockPrefRepo{
		getFn: func(_ context.Context, userID string) ([]*domain.Preference, error) {
			if userID == "user-2" {
				return []*domain.Preference{
					{UserID: "user-2", Channel: domain.ChannelEmail, Enabled: true},
				}, nil
			}
			return nil, nil
		},
	}

	logRepo := &mockLogRepo{
		createFn: func(_ context.Context, log *domain.DeliveryLog) error {
			logEntries = append(logEntries, log)
			return nil
		},
	}

	members := &mockMembers{
		listFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"user-1", "user-2", "user-3"}, nil
		},
	}

	emailSender := &mockSender{
		channel: domain.ChannelEmail,
		sendFn: func(_ context.Context, userID, _, _ string) error {
			emailSentTo = append(emailSentTo, userID)
			return nil
		},
	}

	uc := usecase.New(notifRepo, prefRepo, logRepo, members, emailSender)

	err := uc.Dispatch(t.Context(), domain.NotifyEvent{
		EventType: domain.EventMemberAdded,
		ProjectID: "proj-1",
		ActorID:   "user-1",
		Title:     "New Member",
		Message:   "A new member was added",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Actor (user-1) excluded, so 2 notifications.
	if len(batchCreated) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(batchCreated))
	}
	for _, n := range batchCreated {
		if n.UserID == "user-1" {
			t.Fatal("actor should be excluded from notifications")
		}
	}

	// Only user-2 has email enabled.
	if len(emailSentTo) != 1 || emailSentTo[0] != "user-2" {
		t.Fatalf("expected email sent to [user-2], got %v", emailSentTo)
	}

	// One delivery log entry for user-2's email.
	if len(logEntries) != 1 {
		t.Fatalf("expected 1 delivery log entry, got %d", len(logEntries))
	}
	if logEntries[0].Status != "sent" {
		t.Fatalf("expected status 'sent', got %q", logEntries[0].Status)
	}
}

func TestDispatch_NoMembers(t *testing.T) {
	var batchCalled bool
	notifRepo := &mockNotifRepo{
		createBatchFn: func(_ context.Context, _ []*domain.Notification) error {
			batchCalled = true
			return nil
		},
	}

	members := &mockMembers{
		listFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, nil
		},
	}

	uc := usecase.New(notifRepo, &mockPrefRepo{}, &mockLogRepo{}, members)

	err := uc.Dispatch(t.Context(), domain.NotifyEvent{
		EventType: domain.EventDocumentUploaded,
		ProjectID: "proj-1",
		ActorID:   "user-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if batchCalled {
		t.Fatal("CreateBatch should not be called when there are no recipients")
	}
}

func TestDispatch_OnlyActor(t *testing.T) {
	var batchCalled bool
	notifRepo := &mockNotifRepo{
		createBatchFn: func(_ context.Context, _ []*domain.Notification) error {
			batchCalled = true
			return nil
		},
	}

	members := &mockMembers{
		listFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"actor-1"}, nil
		},
	}

	uc := usecase.New(notifRepo, &mockPrefRepo{}, &mockLogRepo{}, members)

	err := uc.Dispatch(t.Context(), domain.NotifyEvent{
		EventType: domain.EventMemberAdded,
		ProjectID: "proj-1",
		ActorID:   "actor-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if batchCalled {
		t.Fatal("CreateBatch should not be called when only actor is a member")
	}
}

func TestDispatch_MemberListError(t *testing.T) {
	members := &mockMembers{
		listFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, errors.New("db connection failed")
		},
	}

	uc := usecase.New(&mockNotifRepo{}, &mockPrefRepo{}, &mockLogRepo{}, members)

	err := uc.Dispatch(t.Context(), domain.NotifyEvent{
		EventType: domain.EventMemberAdded,
		ProjectID: "proj-1",
		ActorID:   "user-1",
	})
	if err == nil {
		t.Fatal("expected error when member list fails")
	}
}

func TestDispatch_ExternalSendError(t *testing.T) {
	var logEntries []*domain.DeliveryLog

	notifRepo := &mockNotifRepo{}

	prefRepo := &mockPrefRepo{
		getFn: func(_ context.Context, _ string) ([]*domain.Preference, error) {
			return []*domain.Preference{
				{Channel: domain.ChannelEmail, Enabled: true},
			}, nil
		},
	}

	logRepo := &mockLogRepo{
		createFn: func(_ context.Context, log *domain.DeliveryLog) error {
			logEntries = append(logEntries, log)
			return nil
		},
	}

	members := &mockMembers{
		listFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"actor-1", "user-2"}, nil
		},
	}

	failingSender := &mockSender{
		channel: domain.ChannelEmail,
		sendFn: func(_ context.Context, _, _, _ string) error {
			return errors.New("smtp timeout")
		},
	}

	uc := usecase.New(notifRepo, prefRepo, logRepo, members, failingSender)

	// Dispatch should NOT return an error even though external send fails.
	err := uc.Dispatch(t.Context(), domain.NotifyEvent{
		EventType: domain.EventDocumentUploaded,
		ProjectID: "proj-1",
		ActorID:   "actor-1",
		Title:     "Doc Uploaded",
		Message:   "A document was uploaded",
	})
	if err != nil {
		t.Fatalf("Dispatch should not fail on external send error, got: %v", err)
	}

	if len(logEntries) != 1 {
		t.Fatalf("expected 1 delivery log, got %d", len(logEntries))
	}
	if logEntries[0].Status != "failed" {
		t.Fatalf("expected status 'failed', got %q", logEntries[0].Status)
	}
}

func TestList(t *testing.T) {
	expected := []*domain.Notification{
		{ID: "n-1", UserID: "user-1"},
		{ID: "n-2", UserID: "user-1"},
	}

	notifRepo := &mockNotifRepo{
		listFn: func(_ context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected userID: %s", userID)
			}
			if limit != 10 || offset != 0 {
				t.Fatalf("unexpected limit=%d offset=%d", limit, offset)
			}
			return expected, 2, nil
		},
	}

	uc := usecase.New(notifRepo, &mockPrefRepo{}, &mockLogRepo{}, &mockMembers{})

	result, total, err := uc.List(t.Context(), "user-1", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
}

func TestCountUnread(t *testing.T) {
	notifRepo := &mockNotifRepo{
		countUnreadFn: func(_ context.Context, userID string) (int, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected userID: %s", userID)
			}
			return 5, nil
		},
	}

	uc := usecase.New(notifRepo, &mockPrefRepo{}, &mockLogRepo{}, &mockMembers{})

	count, err := uc.CountUnread(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5, got %d", count)
	}
}

func TestMarkRead(t *testing.T) {
	var called bool
	notifRepo := &mockNotifRepo{
		markReadFn: func(_ context.Context, userID, id string) error {
			called = true
			if userID != "user-1" || id != "notif-1" {
				t.Fatalf("unexpected args: userID=%s id=%s", userID, id)
			}
			return nil
		},
	}

	uc := usecase.New(notifRepo, &mockPrefRepo{}, &mockLogRepo{}, &mockMembers{})

	err := uc.MarkRead(t.Context(), "user-1", "notif-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("MarkRead was not called on repo")
	}
}

func TestMarkAllRead(t *testing.T) {
	var called bool
	notifRepo := &mockNotifRepo{
		markAllReadFn: func(_ context.Context, userID string) error {
			called = true
			if userID != "user-1" {
				t.Fatalf("unexpected userID: %s", userID)
			}
			return nil
		},
	}

	uc := usecase.New(notifRepo, &mockPrefRepo{}, &mockLogRepo{}, &mockMembers{})

	err := uc.MarkAllRead(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("MarkAllRead was not called on repo")
	}
}

func TestGetPreferences_WithDefaults(t *testing.T) {
	prefRepo := &mockPrefRepo{
		getFn: func(_ context.Context, _ string) ([]*domain.Preference, error) {
			return []*domain.Preference{
				{UserID: "user-1", Channel: domain.ChannelEmail, Enabled: true},
			}, nil
		},
	}

	uc := usecase.New(&mockNotifRepo{}, prefRepo, &mockLogRepo{}, &mockMembers{})

	prefs, err := uc.GetPreferences(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prefs) != 3 {
		t.Fatalf("expected 3 preferences (with defaults), got %d", len(prefs))
	}

	chanMap := make(map[domain.Channel]bool)
	for _, p := range prefs {
		chanMap[p.Channel] = p.Enabled
	}

	// Email was explicitly set to true.
	if !chanMap[domain.ChannelEmail] {
		t.Fatal("expected email to be enabled")
	}
	// In-app default is enabled.
	if !chanMap[domain.ChannelInApp] {
		t.Fatal("expected in_app to be enabled by default")
	}
	// Telegram default is disabled.
	if chanMap[domain.ChannelTelegram] {
		t.Fatal("expected telegram to be disabled by default")
	}
}

func TestGetPreferences_AllSet(t *testing.T) {
	prefRepo := &mockPrefRepo{
		getFn: func(_ context.Context, _ string) ([]*domain.Preference, error) {
			return []*domain.Preference{
				{UserID: "user-1", Channel: domain.ChannelInApp, Enabled: true},
				{UserID: "user-1", Channel: domain.ChannelEmail, Enabled: true},
				{UserID: "user-1", Channel: domain.ChannelTelegram, Enabled: true},
			}, nil
		},
	}

	uc := usecase.New(&mockNotifRepo{}, prefRepo, &mockLogRepo{}, &mockMembers{})

	prefs, err := uc.GetPreferences(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prefs) != 3 {
		t.Fatalf("expected exactly 3 preferences, got %d", len(prefs))
	}

	// All should be enabled since user explicitly set them.
	for _, p := range prefs {
		if !p.Enabled {
			t.Fatalf("expected channel %s to be enabled", p.Channel)
		}
	}
}

func TestSetPreference_Valid(t *testing.T) {
	var upserted *domain.Preference
	prefRepo := &mockPrefRepo{
		upsertFn: func(_ context.Context, pref *domain.Preference) error {
			upserted = pref
			return nil
		},
	}

	uc := usecase.New(&mockNotifRepo{}, prefRepo, &mockLogRepo{}, &mockMembers{})

	err := uc.SetPreference(t.Context(), "user-1", domain.ChannelEmail, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upserted == nil {
		t.Fatal("expected Upsert to be called")
	}
	if upserted.UserID != "user-1" || upserted.Channel != domain.ChannelEmail || !upserted.Enabled {
		t.Fatalf("unexpected preference: %+v", upserted)
	}
}

func TestSetPreference_InvalidChannel(t *testing.T) {
	uc := usecase.New(&mockNotifRepo{}, &mockPrefRepo{}, &mockLogRepo{}, &mockMembers{})

	err := uc.SetPreference(t.Context(), "user-1", domain.Channel("sms"), true)
	if !errors.Is(err, domain.ErrInvalidChannel) {
		t.Fatalf("expected ErrInvalidChannel, got: %v", err)
	}
}
