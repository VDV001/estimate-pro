package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/auth/domain"
	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/auth/usecase"
	sharedErrors "github.com/daniilrusanov/estimate-pro/backend/internal/shared/errors"
	"github.com/daniilrusanov/estimate-pro/backend/internal/shared/middleware"
	"github.com/daniilrusanov/estimate-pro/backend/pkg/jwt"
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

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtService))
			r.Get("/me", h.Me)
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
	PreferredLocale string `json:"preferred_locale"`
}

func toUserDTO(u *domain.User) userDTO {
	return userDTO{
		ID:              u.ID,
		Email:           u.Email,
		Name:            u.Name,
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

	writeJSON(w, http.StatusCreated, fullAuthResponse{
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

	writeJSON(w, http.StatusOK, fullAuthResponse{
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
		sharedErrors.Unauthorized(w, "invalid refresh token")
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
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

	writeJSON(w, http.StatusOK, toUserDTO(user))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
