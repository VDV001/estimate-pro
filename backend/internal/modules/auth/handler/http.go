// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/usecase"
	sharedErrors "github.com/VDV001/estimate-pro/backend/internal/shared/errors"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
	"github.com/VDV001/estimate-pro/backend/internal/shared/response"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

type Handler struct {
	uc *usecase.AuthUsecase
}

func New(uc *usecase.AuthUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Register(r chi.Router, jwtService *jwt.Service) {
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", h.Login)
		r.Post("/register", h.RegisterUser)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtService))
			r.Get("/me", h.Me)
			r.Patch("/profile", h.UpdateProfile)
			r.Post("/avatar", h.UploadAvatar)
			r.Get("/avatar/{userId}", h.GetAvatar)
			r.Get("/users/search", h.SearchUsers)
			r.Get("/users/colleagues", h.ListColleagues)
		})
	})
}

type fullAuthResponse struct {
	User         userDTO `json:"user"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
}

type userDTO struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	Name            string `json:"name"`
	AvatarURL       string `json:"avatar_url,omitempty"`
	PreferredLocale string `json:"preferred_locale"`
}

func toUserDTO(u *domain.User) userDTO {
	return userDTO{
		ID:              u.ID,
		Email:           u.Email,
		Name:            u.Name,
		AvatarURL:       u.AvatarURL,
		PreferredLocale: u.PreferredLocale,
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
	Name string `json:"name"`
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

	user, err := h.uc.UpdateProfile(r.Context(), usecase.UpdateProfileInput{
		UserID: userID,
		Name:   req.Name,
	})
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

