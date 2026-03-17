package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/auth/domain"
	projectDomain "github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/domain"
	"github.com/daniilrusanov/estimate-pro/backend/pkg/jwt"
)

type AuthUsecase struct {
	userRepo      domain.UserRepository
	workspaceRepo projectDomain.WorkspaceRepository
	jwtService    *jwt.Service
}

func New(userRepo domain.UserRepository, workspaceRepo projectDomain.WorkspaceRepository, jwtService *jwt.Service) *AuthUsecase {
	return &AuthUsecase{userRepo: userRepo, workspaceRepo: workspaceRepo, jwtService: jwtService}
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
	existing, _ := uc.userRepo.GetByEmail(ctx, input.Email)
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

	// Auto-create personal workspace
	workspace := &projectDomain.Workspace{
		ID:        uuid.New().String(),
		Name:      input.Name,
		OwnerID:   user.ID,
		CreatedAt: now,
	}
	if err := uc.workspaceRepo.Create(ctx, workspace); err != nil {
		return nil, fmt.Errorf("auth.Register create workspace: %w", err)
	}

	tokens, err := uc.jwtService.GeneratePair(user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth.Register tokens: %w", err)
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

	return &AuthOutput{User: user, TokenPair: tokens}, nil
}

func (uc *AuthUsecase) Refresh(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := uc.jwtService.ValidateRefresh(refreshToken)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if _, err := uc.userRepo.GetByID(ctx, claims.UserID); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	tokens, err := uc.jwtService.GeneratePair(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth.Refresh: %w", err)
	}

	return tokens, nil
}

func (uc *AuthUsecase) GetCurrentUser(ctx context.Context, userID string) (*domain.User, error) {
	return uc.userRepo.GetByID(ctx, userID)
}
