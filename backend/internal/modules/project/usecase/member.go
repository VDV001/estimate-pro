package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/domain"
)

type MemberUsecase struct {
	memberRepo  domain.MemberRepository
	projectRepo domain.ProjectRepository
	userFinder  domain.UserFinder
}

func NewMemberUsecase(memberRepo domain.MemberRepository, projectRepo domain.ProjectRepository, userFinder domain.UserFinder) *MemberUsecase {
	return &MemberUsecase{memberRepo: memberRepo, projectRepo: projectRepo, userFinder: userFinder}
}

type AddMemberInput struct {
	ProjectID string
	UserID    string
	Role      domain.Role
	CallerID  string
}

func (uc *MemberUsecase) AddMember(ctx context.Context, input AddMemberInput) error {
	if !input.Role.IsValid() {
		return fmt.Errorf("member.AddMember: invalid role %q", input.Role)
	}

	if _, err := uc.projectRepo.GetByID(ctx, input.ProjectID); err != nil {
		return fmt.Errorf("member.AddMember: %w", err)
	}

	callerRole, err := uc.memberRepo.GetRole(ctx, input.ProjectID, input.CallerID)
	if err != nil {
		return fmt.Errorf("member.AddMember: %w", err)
	}
	if !callerRole.CanManageMembers() {
		return domain.ErrInsufficientRole
	}

	member := &domain.Member{
		ProjectID: input.ProjectID,
		UserID:    input.UserID,
		Role:      input.Role,
	}
	if err := uc.memberRepo.Add(ctx, member); err != nil {
		return err
	}
	return nil
}

type AddMemberByEmailInput struct {
	ProjectID string
	Email     string
	Role      domain.Role
	CallerID  string
}

func (uc *MemberUsecase) AddMemberByEmail(ctx context.Context, input AddMemberByEmailInput) error {
	userID, err := uc.userFinder.FindByEmail(ctx, input.Email)
	if err != nil {
		return fmt.Errorf("member.AddMemberByEmail: %w", err)
	}

	return uc.AddMember(ctx, AddMemberInput{
		ProjectID: input.ProjectID,
		UserID:    userID,
		Role:      input.Role,
		CallerID:  input.CallerID,
	})
}

type RemoveMemberInput struct {
	ProjectID string
	UserID    string
	CallerID  string
}

var ErrLastAdmin = errors.New("cannot remove last admin from project")

func (uc *MemberUsecase) RemoveMember(ctx context.Context, input RemoveMemberInput) error {
	callerRole, err := uc.memberRepo.GetRole(ctx, input.ProjectID, input.CallerID)
	if err != nil {
		return fmt.Errorf("member.RemoveMember: %w", err)
	}

	if input.CallerID != input.UserID && !callerRole.CanManageMembers() {
		return domain.ErrInsufficientRole
	}

	targetRole, err := uc.memberRepo.GetRole(ctx, input.ProjectID, input.UserID)
	if err != nil {
		return fmt.Errorf("member.RemoveMember: %w", err)
	}
	if targetRole == domain.RoleAdmin {
		members, err := uc.memberRepo.ListByProject(ctx, input.ProjectID)
		if err != nil {
			return fmt.Errorf("member.RemoveMember: %w", err)
		}
		adminCount := 0
		for _, m := range members {
			if m.Role == domain.RoleAdmin {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return ErrLastAdmin
		}
	}

	return uc.memberRepo.Remove(ctx, input.ProjectID, input.UserID)
}

func (uc *MemberUsecase) ListMembers(ctx context.Context, projectID string) ([]*domain.Member, error) {
	return uc.memberRepo.ListByProject(ctx, projectID)
}

func (uc *MemberUsecase) ListMembersWithUsers(ctx context.Context, projectID string) ([]*domain.MemberWithUser, error) {
	return uc.memberRepo.ListByProjectWithUsers(ctx, projectID)
}
