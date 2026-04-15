package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/handler"
	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/usecase"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

// --- Mock UserRepository ---

type mockUserRepo struct {
	users     map[string]*domain.User // id -> user
	byEmail   map[string]*domain.User // email -> user
	createErr error
	updateErr error
	searchErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:   make(map[string]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, user *domain.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) Update(_ context.Context, user *domain.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *mockUserRepo) Search(_ context.Context, query, excludeUserID string, limit int) ([]*domain.UserSearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	var results []*domain.UserSearchResult
	for _, u := range m.users {
		if u.ID == excludeUserID {
			continue
		}
		results = append(results, &domain.UserSearchResult{
			ID: u.ID, Email: u.Email, Name: u.Name, AvatarURL: u.AvatarURL,
		})
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (m *mockUserRepo) ListColleagues(_ context.Context, userID string, limit int) ([]*domain.UserSearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	var results []*domain.UserSearchResult
	for _, u := range m.users {
		if u.ID == userID {
			continue
		}
		results = append(results, &domain.UserSearchResult{
			ID: u.ID, Email: u.Email, Name: u.Name,
		})
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (m *mockUserRepo) ListRecentlyAdded(_ context.Context, _ string, limit int) ([]*domain.UserSearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return []*domain.UserSearchResult{}, nil
}

func (m *mockUserRepo) seedUser(u *domain.User) {
	m.users[u.ID] = u
	m.byEmail[u.Email] = u
}

// --- Mock WorkspaceCreator ---

type mockWorkspaceCreator struct {
	err error
}

func (m *mockWorkspaceCreator) CreatePersonalWorkspace(_ context.Context, _, _ string) error {
	return m.err
}

// --- Mock TokenStore ---

type mockTokenStore struct {
	tokens    map[string]map[string]bool // userID -> tokenID -> exists
	saveErr   error
	deleteErr error
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]map[string]bool)}
}

func (m *mockTokenStore) Save(_ context.Context, userID, tokenID string, _ time.Duration) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if m.tokens[userID] == nil {
		m.tokens[userID] = make(map[string]bool)
	}
	m.tokens[userID][tokenID] = true
	return nil
}

func (m *mockTokenStore) Exists(_ context.Context, userID, tokenID string) (bool, error) {
	if m.tokens[userID] == nil {
		return false, nil
	}
	return m.tokens[userID][tokenID], nil
}

func (m *mockTokenStore) Delete(_ context.Context, userID, tokenID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if m.tokens[userID] != nil {
		delete(m.tokens[userID], tokenID)
	}
	return nil
}

func (m *mockTokenStore) DeleteAll(_ context.Context, userID string) error {
	delete(m.tokens, userID)
	return nil
}

// --- Mock AvatarStorage ---

type mockAvatarStorage struct {
	data        map[string][]byte
	contentType map[string]string
	uploadErr   error
	downloadErr error
}

func newMockAvatarStorage() *mockAvatarStorage {
	return &mockAvatarStorage{
		data:        make(map[string][]byte),
		contentType: make(map[string]string),
	}
}

func (m *mockAvatarStorage) Upload(_ context.Context, key string, data []byte, ct string) (string, error) {
	if m.uploadErr != nil {
		return "", m.uploadErr
	}
	m.data[key] = data
	m.contentType[key] = ct
	return "/avatars/" + key, nil
}

func (m *mockAvatarStorage) Download(_ context.Context, key string) ([]byte, string, error) {
	if m.downloadErr != nil {
		return nil, "", m.downloadErr
	}
	d, ok := m.data[key]
	if !ok {
		return nil, "", fmt.Errorf("not found")
	}
	return d, m.contentType[key], nil
}

// --- Mock MembershipChecker ---

type mockMembershipChecker struct {
	shared bool
	err    error
}

func (m *mockMembershipChecker) ShareProject(_ context.Context, _, _ string) (bool, error) {
	return m.shared, m.err
}

// --- Mock ResetTokenStore ---

type mockResetTokenStore struct {
	tokens map[string]string // token -> userID
	err    error
}

func newMockResetTokenStore() *mockResetTokenStore {
	return &mockResetTokenStore{tokens: make(map[string]string)}
}

func (m *mockResetTokenStore) Save(_ context.Context, token, userID string, _ time.Duration) error {
	if m.err != nil {
		return m.err
	}
	m.tokens[token] = userID
	return nil
}

func (m *mockResetTokenStore) Consume(_ context.Context, token string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	userID, ok := m.tokens[token]
	if !ok {
		return "", domain.ErrResetTokenNotFound
	}
	delete(m.tokens, token)
	return userID, nil
}

// --- Helpers ---

const testJWTSecret = "test-secret-key-for-handler-tests-1234"

func newJWTService() *jwt.Service {
	return jwt.NewService(testJWTSecret, 15*time.Minute, 7*24*time.Hour)
}

func newTestUsecase(
	userRepo domain.UserRepository,
	wsCreator domain.WorkspaceCreator,
	tokenStore domain.TokenStore,
	avatarStorage domain.AvatarStorage,
	membershipChecker domain.MembershipChecker,
) *usecase.AuthUsecase {
	return usecase.New(userRepo, wsCreator, newJWTService(), tokenStore, avatarStorage, membershipChecker)
}

func newTestHandler(uc *usecase.AuthUsecase) *handler.Handler {
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
// RegisterUser tests
// ==============================

func TestRegisterUser_Success(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{
		"email": "new@example.com", "password": "strongpass123", "name": "New User",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	rec := httptest.NewRecorder()

	h.RegisterUser(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec.Body, &resp)

	if _, ok := resp["access_token"]; !ok {
		t.Error("expected access_token in response")
	}
	if _, ok := resp["refresh_token"]; !ok {
		t.Error("expected refresh_token in response")
	}
	user, ok := resp["user"].(map[string]any)
	if !ok {
		t.Fatal("expected user object in response")
	}
	if user["email"] != "new@example.com" {
		t.Errorf("expected email new@example.com, got %v", user["email"])
	}
}

func TestRegisterUser_InvalidJSON(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()

	h.RegisterUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRegisterUser_MissingFields(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing email", map[string]string{"password": "pass1234", "name": "Name"}},
		{"missing password", map[string]string{"email": "a@b.com", "name": "Name"}},
		{"missing name", map[string]string{"email": "a@b.com", "password": "pass1234"}},
		{"all empty", map[string]string{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", jsonBody(tc.body))
			rec := httptest.NewRecorder()
			h.RegisterUser(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRegisterUser_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "taken@example.com", "Existing", "somehash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{
		"email": "taken@example.com", "password": "strongpass123", "name": "New User",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	rec := httptest.NewRecorder()

	h.RegisterUser(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegisterUser_InputTooLong(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	longStr := make([]byte, 256)
	for i := range longStr {
		longStr[i] = 'a'
	}

	body := jsonBody(map[string]string{
		"email": string(longStr) + "@example.com", "password": "strongpass123", "name": "Name",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	rec := httptest.NewRecorder()

	h.RegisterUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// Login tests
// ==============================

func TestLogin_Success(t *testing.T) {
	repo := newMockUserRepo()
	// bcrypt needs a real hash; Register creates one, so we register first then login.
	tokenStore := newMockTokenStore()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, tokenStore, newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	// Register a user first to get a valid password hash.
	regBody := jsonBody(map[string]string{
		"email": "login@example.com", "password": "password123", "name": "Login User",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", regBody)
	regRec := httptest.NewRecorder()
	h.RegisterUser(regRec, regReq)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register setup failed: %d %s", regRec.Code, regRec.Body.String())
	}

	// Now login.
	body := jsonBody(map[string]string{"email": "login@example.com", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec.Body, &resp)
	if resp["access_token"] == nil {
		t.Error("expected access_token")
	}
	if resp["refresh_token"] == nil {
		t.Error("expected refresh_token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	tokenStore := newMockTokenStore()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, tokenStore, newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	// Register first.
	regBody := jsonBody(map[string]string{
		"email": "wrong@example.com", "password": "correctpass", "name": "User",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", regBody)
	regRec := httptest.NewRecorder()
	h.RegisterUser(regRec, regReq)

	// Login with wrong password.
	body := jsonBody(map[string]string{"email": "wrong@example.com", "password": "wrongpass1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"email": "noone@example.com", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("{")))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLogin_MissingFields(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"email": "a@b.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// ==============================
// Refresh tests
// ==============================

func TestRefresh_Success(t *testing.T) {
	repo := newMockUserRepo()
	tokenStore := newMockTokenStore()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, tokenStore, newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	// Register to get tokens.
	regBody := jsonBody(map[string]string{
		"email": "refresh@example.com", "password": "password123", "name": "Refresh User",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", regBody)
	regRec := httptest.NewRecorder()
	h.RegisterUser(regRec, regReq)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register setup failed: %d", regRec.Code)
	}

	var regResp map[string]any
	decodeJSON(t, regRec.Body, &regResp)
	refreshToken := regResp["refresh_token"].(string)

	// Refresh.
	body := jsonBody(map[string]string{"refresh_token": refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec.Body, &resp)
	if resp["access_token"] == nil {
		t.Error("expected access_token")
	}
	if resp["refresh_token"] == nil {
		t.Error("expected refresh_token")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"refresh_token": "invalid-token"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRefresh_InvalidJSON(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// ==============================
// Logout tests
// ==============================

func TestLogout_Success(t *testing.T) {
	repo := newMockUserRepo()
	tokenStore := newMockTokenStore()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, tokenStore, newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	// Register to get tokens.
	regBody := jsonBody(map[string]string{
		"email": "logout@example.com", "password": "password123", "name": "Logout User",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", regBody)
	regRec := httptest.NewRecorder()
	h.RegisterUser(regRec, regReq)

	var regResp map[string]any
	decodeJSON(t, regRec.Body, &regResp)
	refreshToken := regResp["refresh_token"].(string)

	body := jsonBody(map[string]string{"refresh_token": refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", body)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestLogout_InvalidJSON(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// ==============================
// ForgotPassword tests
// ==============================

func TestForgotPassword_AlwaysReturns200(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"existing email", "exists@example.com"},
		{"non-existing email", "ghost@example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockUserRepo()
			seedTestUserWithHash(repo, "u1", "exists@example.com", "User", "somehash")
			resetStore := newMockResetTokenStore()
			tokenStore := newMockTokenStore()
			uc := newTestUsecase(repo, &mockWorkspaceCreator{}, tokenStore, newMockAvatarStorage(), &mockMembershipChecker{})
			uc.SetResetConfig(resetStore, "http://localhost:3000", 30*time.Minute)
			h := newTestHandler(uc)

			body := jsonBody(map[string]string{"email": tc.email})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", body)
			rec := httptest.NewRecorder()

			h.ForgotPassword(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestForgotPassword_EmptyEmail(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"email": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", body)
	rec := httptest.NewRecorder()

	h.ForgotPassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestForgotPassword_InvalidJSON(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()

	h.ForgotPassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// ==============================
// ResetPassword tests
// ==============================

func TestResetPassword_Success(t *testing.T) {
	repo := newMockUserRepo()
	tokenStore := newMockTokenStore()
	resetStore := newMockResetTokenStore()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, tokenStore, newMockAvatarStorage(), &mockMembershipChecker{})
	uc.SetResetConfig(resetStore, "http://localhost:3000", 30*time.Minute)
	h := newTestHandler(uc)

	// Register user to get a valid bcrypt hash in repo.
	regBody := jsonBody(map[string]string{
		"email": "reset@example.com", "password": "oldpassword1", "name": "Reset User",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", regBody)
	regRec := httptest.NewRecorder()
	h.RegisterUser(regRec, regReq)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register setup failed: %d", regRec.Code)
	}

	// Find the user ID.
	u := repo.byEmail["reset@example.com"]
	resetStore.tokens["valid-token"] = u.ID

	body := jsonBody(map[string]string{"token": "valid-token", "new_password": "newpassword1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", body)
	rec := httptest.NewRecorder()

	h.ResetPasswordHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	repo := newMockUserRepo()
	resetStore := newMockResetTokenStore()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	uc.SetResetConfig(resetStore, "http://localhost:3000", 30*time.Minute)
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"token": "bad-token", "new_password": "newpassword1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", body)
	rec := httptest.NewRecorder()

	h.ResetPasswordHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestResetPassword_ShortPassword(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"token": "some-token", "new_password": "short"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", body)
	rec := httptest.NewRecorder()

	h.ResetPasswordHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestResetPassword_LongPassword(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	longPw := make([]byte, 73)
	for i := range longPw {
		longPw[i] = 'a'
	}
	body := jsonBody(map[string]string{"token": "some-token", "new_password": string(longPw)})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", body)
	rec := httptest.NewRecorder()

	h.ResetPasswordHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestResetPassword_MissingToken(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"token": "", "new_password": "newpassword1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", body)
	rec := httptest.NewRecorder()

	h.ResetPasswordHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestResetPassword_InvalidJSON(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()

	h.ResetPasswordHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// ==============================
// Me tests
// ==============================

func TestMe_Success(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "me@example.com", "Me User", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec.Body, &resp)
	if resp["email"] != "me@example.com" {
		t.Errorf("expected email me@example.com, got %v", resp["email"])
	}
	if resp["name"] != "Me User" {
		t.Errorf("expected name Me User, got %v", resp["name"])
	}
}

func TestMe_NoUserContext(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMe_UserNotFound(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req = requestWithUserID(req, "nonexistent")
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// ==============================
// UpdateProfile tests
// ==============================

func TestUpdateProfile_Success(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "profile@example.com", "Old Name", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", body)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec.Body, &resp)
	if resp["name"] != "New Name" {
		t.Errorf("expected name New Name, got %v", resp["name"])
	}
}

func TestUpdateProfile_InvalidJSON(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "p@example.com", "Name", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader([]byte("bad")))
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateProfile_NoUserContext(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"name": "Name"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", body)
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestUpdateProfile_WithTelegramChatID(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "tg@example.com", "Name", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]any{"telegram_chat_id": "12345"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", body)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec.Body, &resp)
	if resp["telegram_chat_id"] != "12345" {
		t.Errorf("expected telegram_chat_id 12345, got %v", resp["telegram_chat_id"])
	}
}

// ==============================
// SearchUsers tests
// ==============================

func TestSearchUsers_Success(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "caller@example.com", "Caller", "hash")
	seedTestUserWithHash(repo, "u2", "found@example.com", "Found User", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/search?q=found", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.SearchUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp []map[string]any
	decodeJSON(t, rec.Body, &resp)
	// Mock returns all non-caller users; at least one result expected.
	if len(resp) == 0 {
		t.Error("expected at least one search result")
	}
}

func TestSearchUsers_QueryTooShort(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/search?q=a", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.SearchUsers(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSearchUsers_NoUserContext(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/search?q=test", nil)
	rec := httptest.NewRecorder()

	h.SearchUsers(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// ==============================
// ListColleagues tests
// ==============================

func TestListColleagues_Success(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "c@example.com", "Caller", "hash")
	seedTestUserWithHash(repo, "u2", "col@example.com", "Colleague", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/colleagues", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.ListColleagues(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp []map[string]any
	decodeJSON(t, rec.Body, &resp)
	// At minimum should return an array.
	if resp == nil {
		t.Error("expected non-nil response array")
	}
}

func TestListColleagues_NoUserContext(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/colleagues", nil)
	rec := httptest.NewRecorder()

	h.ListColleagues(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// ==============================
// ListRecentlyAdded tests
// ==============================

func TestListRecentlyAdded_Success(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "r@example.com", "Caller", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/recent", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.ListRecentlyAdded(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListRecentlyAdded_NoUserContext(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/recent", nil)
	rec := httptest.NewRecorder()

	h.ListRecentlyAdded(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// ==============================
// UploadAvatar tests
// ==============================

func TestUploadAvatar_Success(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "avatar@example.com", "Avatar User", "hash")
	avatarStore := newMockAvatarStorage()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), avatarStore, &mockMembershipChecker{})
	h := newTestHandler(uc)

	// Build multipart form.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="avatar"; filename="photo.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("fake-image-data"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/avatar", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.UploadAvatar(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec.Body, &resp)
	if resp["avatar_url"] == nil || resp["avatar_url"] == "" {
		t.Error("expected non-empty avatar_url in response")
	}
}

func TestUploadAvatar_NoFile(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "nofile@example.com", "User", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	// Empty multipart form (no file).
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/avatar", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.UploadAvatar(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUploadAvatar_NoUserContext(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/avatar", nil)
	rec := httptest.NewRecorder()

	h.UploadAvatar(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestUploadAvatar_InvalidContentType(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "badct@example.com", "User", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="avatar"; filename="doc.pdf"`},
		"Content-Type":        {"application/pdf"},
	})
	part.Write([]byte("fake-pdf"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/avatar", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.UploadAvatar(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// GetAvatar tests
// ==============================

func TestGetAvatar_OwnAvatar(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "ga@example.com", "User", "hash")
	avatarStore := newMockAvatarStorage()
	avatarStore.data["avatars/u1"] = []byte("image-bytes")
	avatarStore.contentType["avatars/u1"] = "image/png"
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), avatarStore, &mockMembershipChecker{shared: true})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/avatar/u1", nil)
	req = requestWithUserID(req, "u1")
	req = requestWithChiParam(req, "userId", "u1")
	rec := httptest.NewRecorder()

	h.GetAvatar(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected content-type image/png, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != "image-bytes" {
		t.Errorf("expected body 'image-bytes', got %q", rec.Body.String())
	}
}

func TestGetAvatar_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "nf@example.com", "User", "hash")
	avatarStore := newMockAvatarStorage()
	// No avatar stored for u1.
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), avatarStore, &mockMembershipChecker{shared: true})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/avatar/u1", nil)
	req = requestWithUserID(req, "u1")
	req = requestWithChiParam(req, "userId", "u1")
	rec := httptest.NewRecorder()

	h.GetAvatar(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetAvatar_NoUserContext(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/avatar/u1", nil)
	req = requestWithChiParam(req, "userId", "u1")
	rec := httptest.NewRecorder()

	h.GetAvatar(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGetAvatar_OtherUser_NoSharedProject(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "a@example.com", "User A", "hash")
	seedTestUserWithHash(repo, "u2", "b@example.com", "User B", "hash")
	avatarStore := newMockAvatarStorage()
	avatarStore.data["avatars/u2"] = []byte("image")
	avatarStore.contentType["avatars/u2"] = "image/jpeg"
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), avatarStore, &mockMembershipChecker{shared: false})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/avatar/u2", nil)
	req = requestWithUserID(req, "u1")
	req = requestWithChiParam(req, "userId", "u2")
	rec := httptest.NewRecorder()

	h.GetAvatar(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// ==============================
// Error branch tests (internal errors)
// ==============================

func TestRegisterUser_InternalError(t *testing.T) {
	repo := newMockUserRepo()
	repo.createErr = fmt.Errorf("db down")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{
		"email": "fail@example.com", "password": "password123", "name": "Fail User",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	rec := httptest.NewRecorder()

	h.RegisterUser(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLogin_InternalError(t *testing.T) {
	repo := newMockUserRepo()
	// Seed user with empty hash to trigger bcrypt error (invalid hash format).
	seedTestUserWithHash(repo, "u1", "err@example.com", "User", "not-a-bcrypt-hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"email": "err@example.com", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	// bcrypt.CompareHashAndPassword with invalid hash returns ErrHashTooShort → ErrInvalidCredentials
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRefresh_RevokedToken(t *testing.T) {
	repo := newMockUserRepo()
	tokenStore := newMockTokenStore()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, tokenStore, newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	// Register to get tokens.
	regBody := jsonBody(map[string]string{
		"email": "revoked@example.com", "password": "password123", "name": "Revoked User",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", regBody)
	regRec := httptest.NewRecorder()
	h.RegisterUser(regRec, regReq)

	var regResp map[string]any
	decodeJSON(t, regRec.Body, &regResp)
	refreshToken := regResp["refresh_token"].(string)

	// Clear token store to simulate revocation.
	for k := range tokenStore.tokens {
		delete(tokenStore.tokens, k)
	}

	body := jsonBody(map[string]string{"refresh_token": refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchUsers_Error(t *testing.T) {
	repo := newMockUserRepo()
	repo.searchErr = fmt.Errorf("db error")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/search?q=test", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.SearchUsers(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListColleagues_Error(t *testing.T) {
	repo := newMockUserRepo()
	repo.searchErr = fmt.Errorf("db error")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/colleagues", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.ListColleagues(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListRecentlyAdded_Error(t *testing.T) {
	repo := newMockUserRepo()
	repo.searchErr = fmt.Errorf("db error")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/recent", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.ListRecentlyAdded(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestUpdateProfile_UpdateError(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "ue@example.com", "Name", "hash")
	repo.updateErr = fmt.Errorf("db error")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"name": "New"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", body)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestResetPassword_NotConfigured(t *testing.T) {
	repo := newMockUserRepo()
	// No SetResetConfig called → resetTokenStore is nil.
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	body := jsonBody(map[string]string{"token": "some-token", "new_password": "newpassword1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", body)
	rec := httptest.NewRecorder()

	h.ResetPasswordHandler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchUsers_NilResultsReturnsEmptyArray(t *testing.T) {
	repo := newMockUserRepo()
	// No other users seeded, so Search returns nil slice for the caller.
	seedTestUserWithHash(repo, "u1", "only@example.com", "Only", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/search?q=nobody", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.SearchUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify the body contains a JSON array (not null).
	body := rec.Body.String()
	if body == "" || body[0] != '[' {
		t.Errorf("expected JSON array, got %q", body)
	}
}

func TestForgotPassword_RateLimited(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "rl@example.com", "User", "somehash")
	resetStore := newMockResetTokenStore()
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	uc.SetResetConfig(resetStore, "http://localhost:3000", 30*time.Minute)
	h := newTestHandler(uc)

	// The rate limiter allows 3 per 10 min. Hit it 4 times — all should return 200.
	for i := 0; i < 4; i++ {
		body := jsonBody(map[string]string{"email": "rl@example.com"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", body)
		rec := httptest.NewRecorder()
		h.ForgotPassword(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("iteration %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestUploadAvatar_UploadError(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "uploaderr@example.com", "User", "hash")
	avatarStore := newMockAvatarStorage()
	avatarStore.uploadErr = fmt.Errorf("s3 down")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), avatarStore, &mockMembershipChecker{})
	h := newTestHandler(uc)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="avatar"; filename="photo.png"`},
		"Content-Type":        {"image/png"},
	})
	part.Write([]byte("fake-image"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/avatar", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.UploadAvatar(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListColleagues_NilResultsReturnsEmptyArray(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "lc@example.com", "Only", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/colleagues", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.ListColleagues(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body == "" || body[0] != '[' {
		t.Errorf("expected JSON array, got %q", body)
	}
}

func TestListRecentlyAdded_NilResultsReturnsEmptyArray(t *testing.T) {
	repo := newMockUserRepo()
	seedTestUserWithHash(repo, "u1", "lr@example.com", "Only", "hash")
	uc := newTestUsecase(repo, &mockWorkspaceCreator{}, newMockTokenStore(), newMockAvatarStorage(), &mockMembershipChecker{})
	h := newTestHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users/recent", nil)
	req = requestWithUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.ListRecentlyAdded(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body == "" || body[0] != '[' {
		t.Errorf("expected JSON array, got %q", body)
	}
}

// --- Seed helper that does not require bcrypt ---

func seedTestUserWithHash(repo *mockUserRepo, id, email, name, passwordHash string) {
	repo.seedUser(&domain.User{
		ID:              id,
		Email:           email,
		Name:            name,
		PasswordHash:    passwordHash,
		PreferredLocale: "ru",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
}
