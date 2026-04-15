package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetGoogleUser_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/v2/userinfo" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"email":   "alice@google.com",
				"name":    "Alice",
				"picture": "https://lh3.google.com/photo.jpg",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	// Patch the client to point to our mock server.
	client := srv.Client()
	client.Transport = &rewriteTransport{
		base:     http.DefaultTransport,
		original: "https://www.googleapis.com",
		replace:  srv.URL,
	}

	email, name, avatar, err := getGoogleUser(client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "alice@google.com" {
		t.Errorf("email: got %q", email)
	}
	if name != "Alice" {
		t.Errorf("name: got %q", name)
	}
	if avatar != "https://lh3.google.com/photo.jpg" {
		t.Errorf("avatar: got %q", avatar)
	}
}

func TestGetGitHubUser_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			json.NewEncoder(w).Encode(map[string]string{
				"login":      "bob",
				"name":       "Bob Smith",
				"email":      "bob@github.com",
				"avatar_url": "https://avatars.githubusercontent.com/u/123",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{
		base:     http.DefaultTransport,
		original: "https://api.github.com",
		replace:  srv.URL,
	}

	email, name, avatar, err := getGitHubUser(client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "bob@github.com" {
		t.Errorf("email: got %q", email)
	}
	if name != "Bob Smith" {
		t.Errorf("name: got %q", name)
	}
	if avatar != "https://avatars.githubusercontent.com/u/123" {
		t.Errorf("avatar: got %q", avatar)
	}
}

func TestGetGitHubUser_FallbackToLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			json.NewEncoder(w).Encode(map[string]string{
				"login":      "charlie",
				"name":       "", // empty name
				"email":      "charlie@test.com",
				"avatar_url": "",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{
		base:     http.DefaultTransport,
		original: "https://api.github.com",
		replace:  srv.URL,
	}

	_, name, _, err := getGitHubUser(client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "charlie" {
		t.Errorf("name: got %q, want fallback to login 'charlie'", name)
	}
}

func TestGetGitHubUser_FallbackToPrimaryEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			json.NewEncoder(w).Encode(map[string]string{
				"login":      "dave",
				"name":       "Dave",
				"email":      "", // empty email
				"avatar_url": "",
			})
		case "/user/emails":
			json.NewEncoder(w).Encode([]map[string]any{
				{"email": "noreply@github.com", "primary": false, "verified": true},
				{"email": "dave@primary.com", "primary": true, "verified": true},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{
		base:     http.DefaultTransport,
		original: "https://api.github.com",
		replace:  srv.URL,
	}

	email, _, _, err := getGitHubUser(client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "dave@primary.com" {
		t.Errorf("email: got %q, want dave@primary.com", email)
	}
}

func TestGetGitHubUser_NoEmailAnywhere(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			json.NewEncoder(w).Encode(map[string]string{
				"login": "nomail",
				"name":  "No Mail",
				"email": "",
			})
		case "/user/emails":
			json.NewEncoder(w).Encode([]map[string]any{
				{"email": "unverified@test.com", "primary": true, "verified": false},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{
		base:     http.DefaultTransport,
		original: "https://api.github.com",
		replace:  srv.URL,
	}

	_, _, _, err := getGitHubUser(client)
	if err == nil {
		t.Fatal("expected error for no email, got nil")
	}
}

func TestGetUserInfo_UnsupportedProvider(t *testing.T) {
	_, _, _, err := getUserInfo(nil, "twitter", nil)
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

// rewriteTransport rewrites requests to a different base URL for testing.
type rewriteTransport struct {
	base     http.RoundTripper
	original string
	replace  string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite URL from original to replacement.
	url := req.URL.String()
	if len(url) > len(t.original) && url[:len(t.original)] == t.original {
		newURL := t.replace + url[len(t.original):]
		newReq, _ := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
		newReq.Header = req.Header
		return t.base.RoundTrip(newReq)
	}
	return t.base.RoundTrip(req)
}
