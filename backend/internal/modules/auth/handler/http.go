// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/usecase"
	sharedErrors "github.com/VDV001/estimate-pro/backend/internal/shared/errors"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
	"github.com/VDV001/estimate-pro/backend/internal/shared/response"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

type Handler struct {
	uc      *usecase.AuthUsecase
	resetRL *emailRateLimiter
}

func New(uc *usecase.AuthUsecase) *Handler {
	return &Handler{
		uc:      uc,
		resetRL: newEmailRateLimiter(3, time.Hour),
	}
}

func (h *Handler) Register(r chi.Router, jwtService *jwt.Service, rateLimitMW ...func(http.Handler) http.Handler) {
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			if len(rateLimitMW) > 0 {
				r.Use(rateLimitMW[0])
			}
			r.Post("/login", h.Login)
			r.Post("/register", h.RegisterUser)
			r.Post("/refresh", h.Refresh)
			r.Post("/forgot-password", h.ForgotPassword)
			r.Post("/reset-password", h.ResetPasswordHandler)
		})
		r.Post("/logout", h.Logout)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtService))
			r.Get("/me", h.Me)
			r.Patch("/profile", h.UpdateProfile)
			r.Post("/avatar", h.UploadAvatar)
			r.Get("/avatar/{userId}", h.GetAvatar)
			r.Get("/users/search", h.SearchUsers)
			r.Get("/users/colleagues", h.ListColleagues)
			r.Get("/users/recent", h.ListRecentlyAdded)
		})
	})
}

type fullAuthResponse struct {
	User         userDTO `json:"user"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
}

type userDTO struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	AvatarURL         string `json:"avatar_url,omitempty"`
	PreferredLocale   string `json:"preferred_locale"`
	TelegramChatID    string `json:"telegram_chat_id,omitempty"`
	NotificationEmail string `json:"notification_email,omitempty"`
}

func toUserDTO(u *domain.User) userDTO {
	return userDTO{
		ID:                u.ID,
		Email:             u.Email,
		Name:              u.Name,
		AvatarURL:         u.AvatarURL,
		PreferredLocale:   u.PreferredLocale,
		TelegramChatID:    u.TelegramChatID,
		NotificationEmail: u.NotificationEmail,
	}
}

func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		sharedErrors.BadRequest(w, "email, password, and name are required")
		return
	}

	if len(req.Email) > 255 || len(req.Password) > 72 || len(req.Name) > 255 {
		sharedErrors.BadRequest(w, "input too long")
		return
	}

	result, err := h.uc.Register(r.Context(), usecase.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		if errors.Is(err, domain.ErrEmailTaken) {
			sharedErrors.Conflict(w, "email already taken")
			return
		}
		sharedErrors.InternalError(w, "registration failed")
		return
	}

	response.WriteJSON(w, http.StatusCreated, fullAuthResponse{
		User:         toUserDTO(result.User),
		AccessToken:  result.TokenPair.AccessToken,
		RefreshToken: result.TokenPair.RefreshToken,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		sharedErrors.BadRequest(w, "email and password are required")
		return
	}

	result, err := h.uc.Login(r.Context(), usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			sharedErrors.Unauthorized(w, "invalid email or password")
			return
		}
		sharedErrors.InternalError(w, "login failed")
		return
	}

	response.WriteJSON(w, http.StatusOK, fullAuthResponse{
		User:         toUserDTO(result.User),
		AccessToken:  result.TokenPair.AccessToken,
		RefreshToken: result.TokenPair.RefreshToken,
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	tokens, err := h.uc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrTokenRevoked) {
			sharedErrors.Unauthorized(w, "refresh token revoked")
			return
		}
		sharedErrors.Unauthorized(w, "invalid refresh token")
		return
	}

	response.WriteJSON(w, http.StatusOK, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}
	_ = h.uc.Logout(r.Context(), req.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	user, err := h.uc.GetCurrentUser(r.Context(), userID)
	if err != nil {
		sharedErrors.NotFound(w, "user not found")
		return
	}

	response.WriteJSON(w, http.StatusOK, toUserDTO(user))
}

type updateProfileRequest struct {
	Name              string  `json:"name"`
	TelegramChatID    *string `json:"telegram_chat_id,omitempty"`
	NotificationEmail *string `json:"notification_email,omitempty"`
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	input := usecase.UpdateProfileInput{
		UserID: userID,
		Name:   req.Name,
	}
	if req.TelegramChatID != nil {
		input.TelegramChatID = req.TelegramChatID
	}
	if req.NotificationEmail != nil {
		input.NotificationEmail = req.NotificationEmail
	}
	user, err := h.uc.UpdateProfile(r.Context(), input)
	if err != nil {
		sharedErrors.InternalError(w, "failed to update profile")
		return
	}

	response.WriteJSON(w, http.StatusOK, toUserDTO(user))
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		sharedErrors.BadRequest(w, "file too large (max 5MB)")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		sharedErrors.BadRequest(w, "avatar file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		sharedErrors.InternalError(w, "failed to read file")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		sharedErrors.BadRequest(w, "only JPEG, PNG, and WebP images are allowed")
		return
	}

	user, err := h.uc.UploadAvatar(r.Context(), userID, data, contentType)
	if err != nil {
		sharedErrors.InternalError(w, "failed to upload avatar")
		return
	}

	response.WriteJSON(w, http.StatusOK, toUserDTO(user))
}

func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		sharedErrors.BadRequest(w, "query must be at least 2 characters")
		return
	}

	results, err := h.uc.SearchUsers(r.Context(), q, userID, 10)
	if err != nil {
		sharedErrors.InternalError(w, "search failed")
		return
	}

	if results == nil {
		results = make([]*domain.UserSearchResult, 0)
	}
	response.WriteJSON(w, http.StatusOK, results)
}

func (h *Handler) ListColleagues(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	results, err := h.uc.ListColleagues(r.Context(), userID, 20)
	if err != nil {
		sharedErrors.InternalError(w, "failed to list colleagues")
		return
	}

	if results == nil {
		results = make([]*domain.UserSearchResult, 0)
	}
	response.WriteJSON(w, http.StatusOK, results)
}

func (h *Handler) ListRecentlyAdded(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	results, err := h.uc.ListRecentlyAdded(r.Context(), userID, 10)
	if err != nil {
		sharedErrors.InternalError(w, "failed to list recently added users")
		return
	}

	if results == nil {
		results = make([]*domain.UserSearchResult, 0)
	}
	response.WriteJSON(w, http.StatusOK, results)
}

func (h *Handler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}
	targetUserID := chi.URLParam(r, "userId")

	imgData, imgContentType, err := h.uc.GetAvatar(r.Context(), callerID, targetUserID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", imgContentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(imgData)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	if req.Email == "" {
		sharedErrors.BadRequest(w, "email is required")
		return
	}

	// Per-email rate limiting — silent: always return 200 to avoid revealing info.
	if !h.resetRL.Allow(req.Email) {
		response.WriteJSON(w, http.StatusOK, map[string]string{
			"message": "If an account exists, a reset link has been sent",
		})
		return
	}

	_, err := h.uc.ForgotPassword(r.Context(), req.Email)
	if err != nil {
		sharedErrors.InternalError(w, "internal error")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists, a reset link has been sent",
	})
}

func (h *Handler) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	if req.Token == "" {
		sharedErrors.BadRequest(w, "token is required")
		return
	}
	if len(req.NewPassword) < 8 {
		sharedErrors.BadRequest(w, "password must be at least 8 characters")
		return
	}

	err := h.uc.ResetPassword(r.Context(), usecase.ResetPasswordInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		if errors.Is(err, domain.ErrResetTokenNotFound) {
			sharedErrors.BadRequest(w, "invalid or expired reset token")
			return
		}
		sharedErrors.InternalError(w, "failed to reset password")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Password has been reset successfully",
	})
}

// emailRateLimiter limits per-email request frequency (e.g. forgot-password).
type emailRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*emailRateEntry
	limit   int
	window  time.Duration
}

type emailRateEntry struct {
	count   int
	resetAt time.Time
}

func newEmailRateLimiter(limit int, window time.Duration) *emailRateLimiter {
	return &emailRateLimiter{
		entries: make(map[string]*emailRateEntry),
		limit:   limit,
		window:  window,
	}
}

func (rl *emailRateLimiter) Allow(email string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[email]
	if !exists || now.After(entry.resetAt) {
		rl.entries[email] = &emailRateEntry{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if entry.count >= rl.limit {
		return false
	}
	entry.count++
	return true
}

