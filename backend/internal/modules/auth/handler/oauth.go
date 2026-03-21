// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/VDV001/estimate-pro/backend/internal/config"
	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/usecase"
	sharedErrors "github.com/VDV001/estimate-pro/backend/internal/shared/errors"
)

type OAuthHandler struct {
	uc       *usecase.AuthUsecase
	configs  map[string]*oauth2.Config
	frontURL string
}

func NewOAuthHandler(uc *usecase.AuthUsecase, cfg config.OAuthConfig) *OAuthHandler {
	backendCallbackBase := "http://localhost:8080/api/v1/auth/oauth"
	if os.Getenv("OAUTH_BACKEND_URL") != "" {
		backendCallbackBase = os.Getenv("OAUTH_BACKEND_URL") + "/api/v1/auth/oauth"
	}

	configs := make(map[string]*oauth2.Config)

	if cfg.GoogleClientID != "" {
		configs["google"] = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  backendCallbackBase + "/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}
	}

	if cfg.GitHubClientID != "" {
		configs["github"] = &oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURL:  backendCallbackBase + "/github/callback",
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     github.Endpoint,
		}
	}

	return &OAuthHandler{uc: uc, configs: configs, frontURL: cfg.RedirectBaseURL}
}

func (h *OAuthHandler) Register(r chi.Router) {
	r.Route("/api/v1/auth/oauth", func(r chi.Router) {
		r.Get("/{provider}", h.BeginAuth)
		r.Get("/{provider}/callback", h.Callback)
	})
}

// BeginAuth redirects user to OAuth provider.
func (h *OAuthHandler) BeginAuth(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	cfg, ok := h.configs[provider]
	if !ok {
		sharedErrors.BadRequest(w, "unsupported provider: "+provider)
		return
	}

	state := "oauth-state" // TODO: use random state with CSRF protection
	url := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Callback handles OAuth provider callback, exchanges code for token, gets user info.
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	cfg, ok := h.configs[provider]
	if !ok {
		sharedErrors.BadRequest(w, "unsupported provider")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		sharedErrors.BadRequest(w, "missing code parameter")
		return
	}

	// Exchange code for token
	token, err := cfg.Exchange(r.Context(), code)
	if err != nil {
		sharedErrors.InternalError(w, "failed to exchange token")
		return
	}

	// Get user info from provider
	email, name, avatarURL, err := getUserInfo(r.Context(), provider, token)
	if err != nil {
		sharedErrors.InternalError(w, "failed to get user info")
		return
	}

	// Create or link user + generate JWT
	result, err := h.uc.OAuthLogin(r.Context(), usecase.OAuthLoginInput{
		Email:     email,
		Name:      name,
		AvatarURL: avatarURL,
		Provider:  provider,
	})
	if err != nil {
		sharedErrors.InternalError(w, "failed to authenticate")
		return
	}

	// Redirect to frontend with tokens as query params
	redirectURL := fmt.Sprintf("%s/auth/callback?access_token=%s&refresh_token=%s",
		h.frontURL, result.TokenPair.AccessToken, result.TokenPair.RefreshToken)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// getUserInfo fetches email, name, and avatar URL from OAuth provider API.
func getUserInfo(ctx context.Context, provider string, token *oauth2.Token) (email, name, avatarURL string, err error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	switch provider {
	case "google":
		return getGoogleUser(client)
	case "github":
		return getGitHubUser(client)
	default:
		return "", "", "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func getGoogleUser(client *http.Client) (string, string, string, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	var info struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", "", "", err
	}
	return info.Email, info.Name, info.Picture, nil
}

func getGitHubUser(client *http.Client) (string, string, string, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	var user struct {
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &user)

	name := user.Name
	if name == "" {
		name = user.Login
	}

	email := user.Email
	if email == "" {
		email, _ = getGitHubPrimaryEmail(client)
	}

	if email == "" {
		return "", "", "", fmt.Errorf("no email found for GitHub user")
	}

	return email, name, user.AvatarURL, nil
}

func getGitHubPrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	return "", fmt.Errorf("no primary verified email")
}

