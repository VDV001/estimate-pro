// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

type AuthUsecase struct {
	userRepo          domain.UserRepository
	workspaceCreator  domain.WorkspaceCreator
	jwtService        *jwt.Service
	tokenStore        domain.TokenStore
	avatarStorage     domain.AvatarStorage
	membershipChecker domain.MembershipChecker
}

func New(userRepo domain.UserRepository, workspaceCreator domain.WorkspaceCreator, jwtService *jwt.Service, tokenStore domain.TokenStore, avatarStorage domain.AvatarStorage, membershipChecker domain.MembershipChecker) *AuthUsecase {
	return &AuthUsecase{userRepo: userRepo, workspaceCreator: workspaceCreator, jwtService: jwtService, tokenStore: tokenStore, avatarStorage: avatarStorage, membershipChecker: membershipChecker}
}

type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

type AuthOutput struct {
	User      *domain.User
	TokenPair *jwt.TokenPair
}

func (uc *AuthUsecase) Register(ctx context.Context, input RegisterInput) (*AuthOutput, error) {
	existing, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("auth.Register: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("auth.Register hash: %w", err)
	}

	now := time.Now()
	user := &domain.User{
		ID:              uuid.New().String(),
		Email:           input.Email,
		PasswordHash:    string(hash),
		Name:            input.Name,
		PreferredLocale: "ru",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("auth.Register create user: %w", err)
	}

	if err := uc.workspaceCreator.CreatePersonalWorkspace(ctx, user.ID, input.Name); err != nil {
		return nil, fmt.Errorf("auth.Register create workspace: %w", err)
	}

	tokens, err := uc.jwtService.GeneratePair(user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth.Register tokens: %w", err)
	}

	if err := uc.tokenStore.Save(ctx, user.ID, tokens.RefreshID, uc.jwtService.RefreshTTL()); err != nil {
		return nil, fmt.Errorf("auth.Register store token: %w", err)
	}

	return &AuthOutput{User: user, TokenPair: tokens}, nil
}

type LoginInput struct {
	Email    string
	Password string
}

func (uc *AuthUsecase) Login(ctx context.Context, input LoginInput) (*AuthOutput, error) {
	user, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth.Login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	tokens, err := uc.jwtService.GeneratePair(user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth.Login tokens: %w", err)
	}

	if err := uc.tokenStore.Save(ctx, user.ID, tokens.RefreshID, uc.jwtService.RefreshTTL()); err != nil {
		return nil, fmt.Errorf("auth.Login store token: %w", err)
	}

	return &AuthOutput{User: user, TokenPair: tokens}, nil
}

func (uc *AuthUsecase) Refresh(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := uc.jwtService.ValidateRefresh(refreshToken)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	exists, err := uc.tokenStore.Exists(ctx, claims.UserID, claims.ID)
	if err != nil {
		return nil, fmt.Errorf("auth.Refresh check token: %w", err)
	}
	if !exists {
		return nil, domain.ErrTokenRevoked
	}

	if _, err := uc.userRepo.GetByID(ctx, claims.UserID); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Token rotation: delete old, generate new
	_ = uc.tokenStore.Delete(ctx, claims.UserID, claims.ID)

	tokens, err := uc.jwtService.GeneratePair(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth.Refresh generate: %w", err)
	}

	if err := uc.tokenStore.Save(ctx, claims.UserID, tokens.RefreshID, uc.jwtService.RefreshTTL()); err != nil {
		return nil, fmt.Errorf("auth.Refresh store token: %w", err)
	}

	return tokens, nil
}

func (uc *AuthUsecase) Logout(ctx context.Context, refreshToken string) error {
	claims, err := uc.jwtService.ValidateRefresh(refreshToken)
	if err != nil {
		return nil // silently ignore invalid tokens on logout
	}
	return uc.tokenStore.Delete(ctx, claims.UserID, claims.ID)
}

func (uc *AuthUsecase) GetCurrentUser(ctx context.Context, userID string) (*domain.User, error) {
	return uc.userRepo.GetByID(ctx, userID)
}

type UpdateProfileInput struct {
	UserID string
	Name   string
}

func (uc *AuthUsecase) UpdateProfile(ctx context.Context, input UpdateProfileInput) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth.UpdateProfile: %w", err)
	}

	if input.Name != "" {
		user.Name = input.Name
	}
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("auth.UpdateProfile: %w", err)
	}

	return user, nil
}

func (uc *AuthUsecase) UploadAvatar(ctx context.Context, userID string, data []byte, contentType string) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.UploadAvatar: %w", err)
	}

	key := fmt.Sprintf("avatars/%s", userID)
	if _, err := uc.avatarStorage.Upload(ctx, key, data, contentType); err != nil {
		return nil, fmt.Errorf("auth.UploadAvatar upload: %w", err)
	}

	user.AvatarURL = fmt.Sprintf("/api/v1/auth/avatar/%s", userID)
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("auth.UploadAvatar save: %w", err)
	}

	return user, nil
}

func (uc *AuthUsecase) GetAvatar(ctx context.Context, callerID, targetUserID string) ([]byte, string, error) {
	if callerID != targetUserID {
		shared, err := uc.membershipChecker.ShareProject(ctx, callerID, targetUserID)
		if err != nil || !shared {
			return nil, "", fmt.Errorf("auth.GetAvatar: access denied")
		}
	}
	key := fmt.Sprintf("avatars/%s", targetUserID)
	return uc.avatarStorage.Download(ctx, key)
}

func (uc *AuthUsecase) SearchUsers(ctx context.Context, query, callerID string, limit int) ([]*domain.UserSearchResult, error) {
	results, err := uc.userRepo.Search(ctx, query, callerID, limit)
	if err != nil {
		return nil, fmt.Errorf("auth.SearchUsers: %w", err)
	}
	return results, nil
}

func (uc *AuthUsecase) ListColleagues(ctx context.Context, userID string, limit int) ([]*domain.UserSearchResult, error) {
	results, err := uc.userRepo.ListColleagues(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("auth.ListColleagues: %w", err)
	}
	return results, nil
}

type OAuthLoginInput struct {
	Email     string
	Name      string
	AvatarURL string
	Provider  string
}

func (uc *AuthUsecase) OAuthLogin(ctx context.Context, input OAuthLoginInput) (*AuthOutput, error) {
	// Try to find existing user by email
	user, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("auth.OAuthLogin: %w", err)
	}

	if user == nil {
		// Create new user (no password for OAuth users)
		now := time.Now()
		user = &domain.User{
			ID:              uuid.New().String(),
			Email:           input.Email,
			PasswordHash:    "", // OAuth users have no password
			Name:            input.Name,
			AvatarURL:       input.AvatarURL,
			PreferredLocale: "ru",
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := uc.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("auth.OAuthLogin create user: %w", err)
		}

		if err := uc.workspaceCreator.CreatePersonalWorkspace(ctx, user.ID, input.Name); err != nil {
			return nil, fmt.Errorf("auth.OAuthLogin create workspace: %w", err)
		}
	}

	// Update name and avatar from OAuth provider
	changed := false
	if input.Name != "" && user.Name != input.Name {
		user.Name = input.Name
		changed = true
	}
	if input.AvatarURL != "" && user.AvatarURL != input.AvatarURL {
		user.AvatarURL = input.AvatarURL
		changed = true
	}
	if changed {
		user.UpdatedAt = time.Now()
		_ = uc.userRepo.Update(ctx, user)
	}

	tokens, err := uc.jwtService.GeneratePair(user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth.OAuthLogin tokens: %w", err)
	}

	if err := uc.tokenStore.Save(ctx, user.ID, tokens.RefreshID, uc.jwtService.RefreshTTL()); err != nil {
		return nil, fmt.Errorf("auth.OAuthLogin store token: %w", err)
	}

	return &AuthOutput{User: user, TokenPair: tokens}, nil
}
