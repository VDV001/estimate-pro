package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/handler"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/usecase"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
)

// --- Mock NotificationRepository ---

type mockNotifRepo struct {
	notifications map[string][]*domain.Notification // userID -> notifications
	createErr     error
	markReadErr   error
	markAllErr    error
}

func newMockNotifRepo() *mockNotifRepo {
	return &mockNotifRepo{notifications: make(map[string][]*domain.Notification)}
}

func (m *mockNotifRepo) Create(_ context.Context, n *domain.Notification) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.notifications[n.UserID] = append(m.notifications[n.UserID], n)
	return nil
}

func (m *mockNotifRepo) CreateBatch(_ context.Context, notifications []*domain.Notification) error {
	if m.createErr != nil {
		return m.createErr
	}
	for _, n := range notifications {
		m.notifications[n.UserID] = append(m.notifications[n.UserID], n)
	}
	return nil
}

func (m *mockNotifRepo) ListByUser(_ context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error) {
	all := m.notifications[userID]
	total := len(all)
	if offset > total {
		return []*domain.Notification{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *mockNotifRepo) CountUnread(_ context.Context, userID string) (int, error) {
	count := 0
	for _, n := range m.notifications[userID] {
		if !n.Read {
			count++
		}
	}
	return count, nil
}

func (m *mockNotifRepo) MarkRead(_ context.Context, userID, notificationID string) error {
	if m.markReadErr != nil {
		return m.markReadErr
	}
	for _, n := range m.notifications[userID] {
		if n.ID == notificationID {
			n.Read = true
			return nil
		}
	}
	return domain.ErrNotificationNotFound
}

func (m *mockNotifRepo) MarkAllRead(_ context.Context, userID string) error {
	if m.markAllErr != nil {
		return m.markAllErr
	}
	for _, n := range m.notifications[userID] {
		n.Read = true
	}
	return nil
}

func (m *mockNotifRepo) seed(n *domain.Notification) {
	m.notifications[n.UserID] = append(m.notifications[n.UserID], n)
}

// --- Mock PreferenceRepository ---

type mockPrefRepo struct {
	prefs     map[string][]*domain.Preference // userID -> prefs
	upsertErr error
}

func newMockPrefRepo() *mockPrefRepo {
	return &mockPrefRepo{prefs: make(map[string][]*domain.Preference)}
}

func (m *mockPrefRepo) Get(_ context.Context, userID string) ([]*domain.Preference, error) {
	return m.prefs[userID], nil
}

func (m *mockPrefRepo) Upsert(_ context.Context, pref *domain.Preference) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	existing := m.prefs[pref.UserID]
	for i, p := range existing {
		if p.Channel == pref.Channel {
			existing[i] = pref
			return nil
		}
	}
	m.prefs[pref.UserID] = append(existing, pref)
	return nil
}

// --- Mock DeliveryLogRepository ---

type mockDeliveryLogRepo struct{}

func (m *mockDeliveryLogRepo) Create(_ context.Context, _ *domain.DeliveryLog) error {
	return nil
}

// --- Mock MemberLister ---

type mockMemberLister struct {
	members map[string][]string // projectID -> userIDs
}

func (m *mockMemberLister) ListMemberUserIDs(_ context.Context, projectID string) ([]string, error) {
	return m.members[projectID], nil
}

// --- Helpers ---

const testUserID = "user-001"

func newTestUsecase(notifRepo domain.NotificationRepository, prefRepo domain.PreferenceRepository) *usecase.NotificationUsecase {
	return usecase.New(notifRepo, prefRepo, &mockDeliveryLogRepo{}, &mockMemberLister{members: make(map[string][]string)})
}

func newTestHandler(uc *usecase.NotificationUsecase) *handler.Handler {
	return handler.New(uc)
}

func requestWithUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func requestWithChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func jsonBody(v any) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func decodeJSON(t *testing.T, body io.Reader, v any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
}

// ==============================
// List tests
// ==============================

func TestList_Paginated(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()

	for i := range 5 {
		notifRepo.seed(&domain.Notification{
			ID: "notif-" + string(rune('a'+i)), UserID: testUserID,
			EventType: domain.EventMemberAdded, Title: "Test",
			Message: "test message", Read: false, CreatedAt: time.Now(),
		})
	}

	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?page=1&limit=3", nil)
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec.Body, &resp)

	notifications, ok := resp["notifications"].([]any)
	if !ok {
		t.Fatal("expected notifications array in response")
	}
	if len(notifications) != 3 {
		t.Errorf("expected 3 notifications (limit=3), got %d", len(notifications))
	}

	meta, ok := resp["meta"].(map[string]any)
	if !ok {
		t.Fatal("expected meta object in response")
	}
	if int(meta["total"].(float64)) != 5 {
		t.Errorf("expected total=5, got %v", meta["total"])
	}
}

func TestList_MissingUserContext(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestList_Empty(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// UnreadCount tests
// ==============================

func TestUnreadCount_Success(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	notifRepo.seed(&domain.Notification{
		ID: "notif-1", UserID: testUserID, EventType: domain.EventMemberAdded,
		Title: "Test", Message: "msg", Read: false, CreatedAt: time.Now(),
	})
	notifRepo.seed(&domain.Notification{
		ID: "notif-2", UserID: testUserID, EventType: domain.EventMemberAdded,
		Title: "Test 2", Message: "msg", Read: true, CreatedAt: time.Now(),
	})

	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.UnreadCount(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]int
	decodeJSON(t, rec.Body, &resp)
	if resp["count"] != 1 {
		t.Errorf("expected unread count 1, got %d", resp["count"])
	}
}

func TestUnreadCount_MissingUserContext(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	rec := httptest.NewRecorder()

	h.UnreadCount(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// MarkRead tests
// ==============================

func TestMarkRead_Success(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	notifRepo.seed(&domain.Notification{
		ID: "notif-1", UserID: testUserID, EventType: domain.EventMemberAdded,
		Title: "Test", Message: "msg", Read: false, CreatedAt: time.Now(),
	})

	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/notif-1/read", nil)
	req = requestWithChiParam(req, "id", "notif-1")
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.MarkRead(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMarkRead_NotFound(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/nonexistent/read", nil)
	req = requestWithChiParam(req, "id", "nonexistent")
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.MarkRead(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMarkRead_MissingID(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications//read", nil)
	req = requestWithChiParam(req, "id", "")
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.MarkRead(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMarkRead_MissingUserContext(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/notif-1/read", nil)
	req = requestWithChiParam(req, "id", "notif-1")
	rec := httptest.NewRecorder()

	h.MarkRead(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// MarkAllRead tests
// ==============================

func TestMarkAllRead_Success(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	notifRepo.seed(&domain.Notification{
		ID: "notif-1", UserID: testUserID, EventType: domain.EventMemberAdded,
		Title: "Test", Message: "msg", Read: false, CreatedAt: time.Now(),
	})
	notifRepo.seed(&domain.Notification{
		ID: "notif-2", UserID: testUserID, EventType: domain.EventDocumentUploaded,
		Title: "Test 2", Message: "msg", Read: false, CreatedAt: time.Now(),
	})

	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/read-all", nil)
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.MarkAllRead(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMarkAllRead_MissingUserContext(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/read-all", nil)
	rec := httptest.NewRecorder()

	h.MarkAllRead(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// GetPreferences tests
// ==============================

func TestGetPreferences_Success(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	prefRepo.prefs[testUserID] = []*domain.Preference{
		{UserID: testUserID, Channel: domain.ChannelInApp, Enabled: true},
	}

	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/preferences", nil)
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.GetPreferences(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec.Body, &resp)
	prefs, ok := resp["preferences"].([]any)
	if !ok {
		t.Fatal("expected preferences array in response")
	}
	// Should have 3 channels (in_app from DB + email/telegram defaults).
	if len(prefs) != 3 {
		t.Errorf("expected 3 preferences (with defaults), got %d", len(prefs))
	}
}

func TestGetPreferences_DefaultsWhenEmpty(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/preferences", nil)
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.GetPreferences(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec.Body, &resp)
	prefs, ok := resp["preferences"].([]any)
	if !ok {
		t.Fatal("expected preferences array in response")
	}
	if len(prefs) != 3 {
		t.Errorf("expected 3 default preferences, got %d", len(prefs))
	}
}

func TestGetPreferences_MissingUserContext(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/preferences", nil)
	rec := httptest.NewRecorder()

	h.GetPreferences(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// SetPreference tests
// ==============================

func TestSetPreference_ValidChannel(t *testing.T) {
	channels := []string{"in_app", "email", "telegram"}

	for _, ch := range channels {
		t.Run(ch, func(t *testing.T) {
			notifRepo := newMockNotifRepo()
			prefRepo := newMockPrefRepo()
			uc := newTestUsecase(notifRepo, prefRepo)
			h := newTestHandler(uc)

			body := jsonBody(map[string]any{"channel": ch, "enabled": true})
			req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", body)
			req = requestWithUserID(req, testUserID)
			rec := httptest.NewRecorder()

			h.SetPreference(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestSetPreference_InvalidChannel(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	body := jsonBody(map[string]any{"channel": "sms", "enabled": true})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", body)
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.SetPreference(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSetPreference_InvalidJSON(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader([]byte("not json")))
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.SetPreference(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSetPreference_MissingUserContext(t *testing.T) {
	notifRepo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	uc := newTestUsecase(notifRepo, prefRepo)
	h := newTestHandler(uc)

	body := jsonBody(map[string]any{"channel": "email", "enabled": true})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", body)
	rec := httptest.NewRecorder()

	h.SetPreference(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}
