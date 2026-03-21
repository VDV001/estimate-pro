package repository

import (
	"context"
	"fmt"

	authDomain "github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
)

// UserFinderAdapter adapts auth.UserRepository to project.domain.UserFinder.
type UserFinderAdapter struct {
	userRepo authDomain.UserRepository
}

func NewUserFinderAdapter(userRepo authDomain.UserRepository) *UserFinderAdapter {
	return &UserFinderAdapter{userRepo: userRepo}
}

func (a *UserFinderAdapter) FindByEmail(ctx context.Context, email string) (string, error) {
	user, err := a.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("userFinder.FindByEmail: %w", err)
	}
	return user.ID, nil
}
