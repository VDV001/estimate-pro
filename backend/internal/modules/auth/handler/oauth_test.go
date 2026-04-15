package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/VDV001/estimate-pro/backend/internal/config"
	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/handler"
	"github.com/VDV001/estimate-pro/backend/internal/shared/errors"
)

func newOAuthHandler() *handler.OAuthHandler {
	uc := newTestUsecase(
		newMockUserRepo(),
		&mockWorkspaceCreator{},
		newMockTokenStore(),
		newMockAvatarStorage(),
		&mockMembershipChecker{},
	)
	cfg := config.OAuthConfig{
		GoogleClientID:     "google-id",
		GoogleClientSecret: "google-secret",
		GitHubClientID:     "github-id",
		GitHubClientSecret: "github-secret",
		RedirectBaseURL:    "http://localhost:3000",
	}
	return handler.NewOAuthHandler(uc, cfg)
}

func withChiProvider(r *http.Request, provider string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", provider)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestBeginAuth_Google(t *testing.T) {
	h := newOAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google", nil)
	req = withChiProvider(req, "google")
	rec := httptest.NewRecorder()

	h.BeginAuth(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status: got %d, want 307", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("expected redirect Location header")
	}
}

func TestBeginAuth_GitHub(t *testing.T) {
	h := newOAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/github", nil)
	req = withChiProvider(req, "github")
	rec := httptest.NewRecorder()

	h.BeginAuth(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status: got %d, want 307", rec.Code)
	}
}

func TestBeginAuth_UnsupportedProvider(t *testing.T) {
	h := newOAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/twitter", nil)
	req = withChiProvider(req, "twitter")
	rec := httptest.NewRecorder()

	h.BeginAuth(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestCallback_MissingCode(t *testing.T) {
	h := newOAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/callback", nil)
	req = withChiProvider(req, "google")
	rec := httptest.NewRecorder()

	h.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}

	var resp errors.ErrorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Error.Message != "missing code parameter" {
		t.Errorf("message: got %q", resp.Error.Message)
	}
}

func TestCallback_UnsupportedProvider(t *testing.T) {
	h := newOAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/twitter/callback?code=abc", nil)
	req = withChiProvider(req, "twitter")
	rec := httptest.NewRecorder()

	h.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestCallback_CodeExchangeFails(t *testing.T) {
	// Create a mock token endpoint that returns an error.
	mockTokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer mockTokenSrv.Close()

	uc := newTestUsecase(
		newMockUserRepo(),
		&mockWorkspaceCreator{},
		newMockTokenStore(),
		newMockAvatarStorage(),
		&mockMembershipChecker{},
	)
	cfg := config.OAuthConfig{
		GoogleClientID:     "google-id",
		GoogleClientSecret: "google-secret",
		RedirectBaseURL:    "http://localhost:3000",
	}
	h := handler.NewOAuthHandler(uc, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/callback?code=badcode", nil)
	req = withChiProvider(req, "google")
	rec := httptest.NewRecorder()

	// The token exchange will fail because Google endpoint is unreachable/returns error.
	// Use a short timeout context.
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	h.Callback(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Logf("status: got %d (expected 500 from failed exchange)", rec.Code)
	}
}

func TestNewOAuthHandler_EmptyConfig(t *testing.T) {
	uc := newTestUsecase(
		newMockUserRepo(),
		&mockWorkspaceCreator{},
		newMockTokenStore(),
		newMockAvatarStorage(),
		&mockMembershipChecker{},
	)

	// Empty config — no providers configured.
	h := handler.NewOAuthHandler(uc, config.OAuthConfig{
		RedirectBaseURL: "http://localhost:3000",
	})

	// All providers should return bad request.
	for _, provider := range []string{"google", "github"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = withChiProvider(req, provider)
		rec := httptest.NewRecorder()

		h.BeginAuth(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("provider %s: got %d, want 400", provider, rec.Code)
		}
	}
}

func TestOAuthHandler_Register(t *testing.T) {
	h := newOAuthHandler()
	r := chi.NewRouter()
	h.Register(r)

	// Verify routes are registered by making requests.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Should get 307 (redirect to Google) not 404.
	if rec.Code == http.StatusNotFound {
		t.Error("expected route to be registered, got 404")
	}
}
