package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/project/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/project/usecase"
)

// --- Mock repositories ---

type mockMemberRepo struct {
	members []*domain.Member
}

func (m *mockMemberRepo) Add(_ context.Context, member *domain.Member) error {
	for _, existing := range m.members {
		if existing.ProjectID == member.ProjectID && existing.UserID == member.UserID {
			return domain.ErrMemberAlreadyAdded
		}
	}
	m.members = append(m.members, member)
	return nil
}

func (m *mockMemberRepo) Remove(_ context.Context, projectID, userID string) error {
	for i, member := range m.members {
		if member.ProjectID == projectID && member.UserID == userID {
			m.members = append(m.members[:i], m.members[i+1:]...)
			return nil
		}
	}
	return domain.ErrMemberNotFound
}

func (m *mockMemberRepo) ListByProject(_ context.Context, projectID string) ([]*domain.Member, error) {
	var result []*domain.Member
	for _, member := range m.members {
		if member.ProjectID == projectID {
			result = append(result, member)
		}
	}
	return result, nil
}

func (m *mockMemberRepo) ListByProjectWithUsers(_ context.Context, projectID string) ([]*domain.MemberWithUser, error) {
	var result []*domain.MemberWithUser
	for _, member := range m.members {
		if member.ProjectID == projectID {
			result = append(result, &domain.MemberWithUser{
				ProjectID: member.ProjectID,
				UserID:    member.UserID,
				Role:      member.Role,
				UserName:  "Test User",
				UserEmail: "test@example.com",
			})
		}
	}
	return result, nil
}

func (m *mockMemberRepo) GetRole(_ context.Context, projectID, userID string) (domain.Role, error) {
	for _, member := range m.members {
		if member.ProjectID == projectID && member.UserID == userID {
			return member.Role, nil
		}
	}
	return "", domain.ErrMemberNotFound
}

type mockProjectRepo struct {
	projects map[string]*domain.Project
}

func (m *mockProjectRepo) Create(_ context.Context, p *domain.Project) error {
	m.projects[p.ID] = p
	return nil
}
func (m *mockProjectRepo) GetByID(_ context.Context, id string) (*domain.Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, domain.ErrProjectNotFound
	}
	return p, nil
}
func (m *mockProjectRepo) ListByWorkspace(_ context.Context, _ string, _, _ int) ([]*domain.Project, int, error) {
	return nil, 0, nil
}
func (m *mockProjectRepo) ListByUser(_ context.Context, _ string, _, _ int) ([]*domain.Project, int, error) {
	return nil, 0, nil
}
func (m *mockProjectRepo) Update(_ context.Context, _ *domain.Project) error { return nil }
func (m *mockProjectRepo) Delete(_ context.Context, _ string) error          { return nil }

// --- Tests ---

func TestMemberUsecase_AddMember(t *testing.T) {
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "admin-1", Role: domain.RoleAdmin},
		},
	}
	projectRepo := &mockProjectRepo{
		projects: map[string]*domain.Project{
			"p-1": {ID: "p-1", WorkspaceID: "ws-1"},
		},
	}
	uc := usecase.NewMemberUsecase(memberRepo, projectRepo, nil)

	t.Run("admin can add member", func(t *testing.T) {
		err := uc.AddMember(context.Background(), usecase.AddMemberInput{
			ProjectID: "p-1", UserID: "dev-1", Role: domain.RoleDeveloper, CallerID: "admin-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("developer cannot add member", func(t *testing.T) {
		memberRepo.members = append(memberRepo.members, &domain.Member{
			ProjectID: "p-1", UserID: "dev-1", Role: domain.RoleDeveloper,
		})
		err := uc.AddMember(context.Background(), usecase.AddMemberInput{
			ProjectID: "p-1", UserID: "dev-2", Role: domain.RoleDeveloper, CallerID: "dev-1",
		})
		if err != domain.ErrInsufficientRole {
			t.Fatalf("expected ErrInsufficientRole, got: %v", err)
		}
	})

	t.Run("duplicate member returns error", func(t *testing.T) {
		err := uc.AddMember(context.Background(), usecase.AddMemberInput{
			ProjectID: "p-1", UserID: "dev-1", Role: domain.RoleDeveloper, CallerID: "admin-1",
		})
		if !errors.Is(err, domain.ErrMemberAlreadyAdded) {
			t.Fatalf("expected ErrMemberAlreadyAdded, got: %v", err)
		}
	})

	t.Run("invalid role returns error", func(t *testing.T) {
		err := uc.AddMember(context.Background(), usecase.AddMemberInput{
			ProjectID: "p-1", UserID: "dev-3", Role: "superadmin", CallerID: "admin-1",
		})
		if err == nil {
			t.Fatal("expected error for invalid role")
		}
	})
}

func TestMemberUsecase_RemoveMember(t *testing.T) {
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "admin-1", Role: domain.RoleAdmin},
			{ProjectID: "p-1", UserID: "dev-1", Role: domain.RoleDeveloper},
		},
	}
	projectRepo := &mockProjectRepo{
		projects: map[string]*domain.Project{"p-1": {ID: "p-1"}},
	}
	uc := usecase.NewMemberUsecase(memberRepo, projectRepo, nil)

	t.Run("admin can remove member", func(t *testing.T) {
		err := uc.RemoveMember(context.Background(), usecase.RemoveMemberInput{
			ProjectID: "p-1", UserID: "dev-1", CallerID: "admin-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("cannot remove self if last admin", func(t *testing.T) {
		err := uc.RemoveMember(context.Background(), usecase.RemoveMemberInput{
			ProjectID: "p-1", UserID: "admin-1", CallerID: "admin-1",
		})
		if err == nil {
			t.Fatal("expected error when removing last admin")
		}
	})
}

// --- Mock UserFinder ---

type mockUserFinder struct {
	emails map[string]string // email -> userID
}

func (m *mockUserFinder) FindByEmail(_ context.Context, email string) (string, error) {
	id, ok := m.emails[email]
	if !ok {
		return "", errors.New("user not found")
	}
	return id, nil
}

func TestMemberUsecase_AddMemberByEmail(t *testing.T) {
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "admin-1", Role: domain.RoleAdmin},
		},
	}
	projectRepo := &mockProjectRepo{
		projects: map[string]*domain.Project{
			"p-1": {ID: "p-1", WorkspaceID: "ws-1"},
		},
	}
	userFinder := &mockUserFinder{
		emails: map[string]string{"dev@example.com": "dev-1"},
	}
	uc := usecase.NewMemberUsecase(memberRepo, projectRepo, userFinder)

	t.Run("success", func(t *testing.T) {
		err := uc.AddMemberByEmail(t.Context(), usecase.AddMemberByEmailInput{
			ProjectID: "p-1",
			Email:     "dev@example.com",
			Role:      domain.RoleDeveloper,
			CallerID:  "admin-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("user not found by email", func(t *testing.T) {
		err := uc.AddMemberByEmail(t.Context(), usecase.AddMemberByEmailInput{
			ProjectID: "p-1",
			Email:     "nonexistent@example.com",
			Role:      domain.RoleDeveloper,
			CallerID:  "admin-1",
		})
		if err == nil {
			t.Fatal("expected error for non-existent email")
		}
	})
}

func TestMemberUsecase_ListMembersWithUsers(t *testing.T) {
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "admin-1", Role: domain.RoleAdmin},
			{ProjectID: "p-1", UserID: "dev-1", Role: domain.RoleDeveloper},
		},
	}
	projectRepo := &mockProjectRepo{
		projects: map[string]*domain.Project{"p-1": {ID: "p-1"}},
	}
	uc := usecase.NewMemberUsecase(memberRepo, projectRepo, nil)

	members, err := uc.ListMembersWithUsers(t.Context(), "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].UserName != "Test User" {
		t.Errorf("expected 'Test User', got %q", members[0].UserName)
	}
}

func TestMemberUsecase_ListMembers(t *testing.T) {
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "admin-1", Role: domain.RoleAdmin},
			{ProjectID: "p-1", UserID: "dev-1", Role: domain.RoleDeveloper},
			{ProjectID: "p-2", UserID: "other", Role: domain.RoleObserver},
		},
	}
	projectRepo := &mockProjectRepo{
		projects: map[string]*domain.Project{"p-1": {ID: "p-1"}},
	}
	uc := usecase.NewMemberUsecase(memberRepo, projectRepo, nil)

	members, err := uc.ListMembers(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}
