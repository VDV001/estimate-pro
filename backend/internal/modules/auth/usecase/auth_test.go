package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

// --- Mock UserRepository ---

type mockUserRepo struct {
	createFn     func(ctx context.Context, user *domain.User) error
	getByIDFn    func(ctx context.Context, id string) (*domain.User, error)
	getByEmailFn func(ctx context.Context, email string) (*domain.User, error)
	updateFn     func(ctx context.Context, user *domain.User) error
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil
}

// --- Mock WorkspaceRepository ---

type mockWorkspaceCreator struct {
	createFn func(ctx context.Context, userID, name string) error
}

func (m *mockWorkspaceCreator) CreatePersonalWorkspace(ctx context.Context, userID, name string) error {
	if m.createFn != nil {
		return m.createFn(ctx, userID, name)
	}
	return nil
}

type mockTokenStore struct {
	tokens map[string]bool
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]bool)}
}

func (m *mockTokenStore) Save(_ context.Context, userID, tokenID string, _ time.Duration) error {
	m.tokens[userID+":"+tokenID] = true
	return nil
}

func (m *mockTokenStore) Exists(_ context.Context, userID, tokenID string) (bool, error) {
	return m.tokens[userID+":"+tokenID], nil
}

func (m *mockTokenStore) Delete(_ context.Context, userID, tokenID string) error {
	delete(m.tokens, userID+":"+tokenID)
	return nil
}

func (m *mockTokenStore) DeleteAll(_ context.Context, userID string) error {
	for k := range m.tokens {
		if len(k) > len(userID) && k[:len(userID)+1] == userID+":" {
			delete(m.tokens, k)
		}
	}
	return nil
}

type mockAvatarStorage struct{}

func (m *mockAvatarStorage) Upload(_ context.Context, key string, _ []byte, _ string) (string, error) {
	return "/avatars/" + key, nil
}

func (m *mockAvatarStorage) Download(_ context.Context, _ string) ([]byte, string, error) {
	return []byte("fake-image"), "image/jpeg", nil
}

type mockMembershipChecker struct{}

func (m *mockMembershipChecker) ShareProject(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

// --- Helper ---

func newTestJWT() *jwt.Service {
	return jwt.NewService("test-secret-key-for-unit-tests", 15*time.Minute, 7*24*time.Hour)
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hash)
}

// --- Tests ---

func TestRegister(t *testing.T) {
	tests := []struct {
		name      string
		input     RegisterInput
		userRepo  *mockUserRepo
		wsRepo    *mockWorkspaceCreator
		wantErr   error
		wantUser  bool
		wantToken bool
	}{
		{
			name: "Success",
			input: RegisterInput{
				Email:    "alice@example.com",
				Password: "strongpassword",
				Name:     "Alice",
			},
			userRepo: &mockUserRepo{
				getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, domain.ErrUserNotFound
				},
				createFn: func(_ context.Context, _ *domain.User) error {
					return nil
				},
			},
			wsRepo: &mockWorkspaceCreator{
				createFn: func(_ context.Context, _, _ string) error {
					return nil
				},
			},
			wantErr:   nil,
			wantUser:  true,
			wantToken: true,
		},
		{
			name: "DuplicateEmail",
			input: RegisterInput{
				Email:    "taken@example.com",
				Password: "password123",
				Name:     "Bob",
			},
			userRepo: &mockUserRepo{
				getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
					return &domain.User{ID: "existing-id", Email: "taken@example.com"}, nil
				},
			},
			wsRepo:    &mockWorkspaceCreator{},
			wantErr:   domain.ErrEmailTaken,
			wantUser:  false,
			wantToken: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := New(tc.userRepo, tc.wsRepo, newTestJWT(), newMockTokenStore(), &mockAvatarStorage{}, &mockMembershipChecker{})

			result, err := uc.Register(t.Context(), tc.input)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				if result != nil {
					t.Fatal("expected nil result on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantUser && result.User == nil {
				t.Fatal("expected user in result")
			}
			if tc.wantUser {
				if result.User.Email != tc.input.Email {
					t.Errorf("expected email %q, got %q", tc.input.Email, result.User.Email)
				}
				if result.User.Name != tc.input.Name {
					t.Errorf("expected name %q, got %q", tc.input.Name, result.User.Name)
				}
				if result.User.ID == "" {
					t.Error("expected non-empty user ID")
				}
			}
			if tc.wantToken {
				if result.TokenPair == nil {
					t.Fatal("expected token pair")
				}
				if result.TokenPair.AccessToken == "" {
					t.Error("expected non-empty access token")
				}
				if result.TokenPair.RefreshToken == "" {
					t.Error("expected non-empty refresh token")
				}
			}
		})
	}
}

func TestLogin(t *testing.T) {
	const testPassword = "correct-password"

	tests := []struct {
		name    string
		input   LoginInput
		repo    *mockUserRepo
		wantErr error
	}{
		{
			name: "Success",
			input: LoginInput{
				Email:    "alice@example.com",
				Password: testPassword,
			},
			repo: &mockUserRepo{
				getByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
					return &domain.User{
						ID:           "user-1",
						Email:        email,
						PasswordHash: hashPassword(t, testPassword),
						Name:         "Alice",
					}, nil
				},
			},
			wantErr: nil,
		},
		{
			name: "WrongPassword",
			input: LoginInput{
				Email:    "alice@example.com",
				Password: "wrong-password",
			},
			repo: &mockUserRepo{
				getByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
					return &domain.User{
						ID:           "user-1",
						Email:        email,
						PasswordHash: hashPassword(t, testPassword),
						Name:         "Alice",
					}, nil
				},
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name: "UserNotFound",
			input: LoginInput{
				Email:    "nobody@example.com",
				Password: "anything",
			},
			repo: &mockUserRepo{
				getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, domain.ErrUserNotFound
				},
			},
			wantErr: domain.ErrInvalidCredentials,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := New(tc.repo, &mockWorkspaceCreator{}, newTestJWT(), newMockTokenStore(), &mockAvatarStorage{}, &mockMembershipChecker{})

			result, err := uc.Login(t.Context(), tc.input)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				if result != nil {
					t.Fatal("expected nil result on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.User == nil {
				t.Fatal("expected user in result")
			}
			if result.User.Email != tc.input.Email {
				t.Errorf("expected email %q, got %q", tc.input.Email, result.User.Email)
			}
			if result.TokenPair == nil {
				t.Fatal("expected token pair")
			}
			if result.TokenPair.AccessToken == "" {
				t.Error("expected non-empty access token")
			}
			if result.TokenPair.RefreshToken == "" {
				t.Error("expected non-empty refresh token")
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	jwtSvc := newTestJWT()

	// Generate a valid refresh token for testing
	validPair, err := jwtSvc.GeneratePair("user-123")
	if err != nil {
		t.Fatalf("failed to generate test tokens: %v", err)
	}

	tests := []struct {
		name         string
		refreshToken string
		repo         *mockUserRepo
		wantErr      error
	}{
		{
			name:         "Success",
			refreshToken: validPair.RefreshToken,
			repo: &mockUserRepo{
				getByIDFn: func(_ context.Context, id string) (*domain.User, error) {
					if id == "user-123" {
						return &domain.User{ID: id, Email: "alice@example.com"}, nil
					}
					return nil, domain.ErrUserNotFound
				},
			},
			wantErr: nil,
		},
		{
			name:         "InvalidToken",
			refreshToken: "not-a-valid-jwt-token",
			repo:         &mockUserRepo{},
			wantErr:      domain.ErrInvalidCredentials,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newMockTokenStore()
			// Pre-save the valid token so Refresh finds it
			if tc.wantErr == nil {
				store.Save(t.Context(), "user-123", validPair.RefreshID, 7*24*time.Hour)
			}
			uc := New(tc.repo, &mockWorkspaceCreator{}, jwtSvc, store, &mockAvatarStorage{}, &mockMembershipChecker{})

			tokens, err := uc.Refresh(t.Context(), tc.refreshToken)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				if tokens != nil {
					t.Fatal("expected nil tokens on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokens == nil {
				t.Fatal("expected non-nil tokens")
			}
			if tokens.AccessToken == "" {
				t.Error("expected non-empty access token")
			}
			if tokens.RefreshToken == "" {
				t.Error("expected non-empty refresh token")
			}
		})
	}
}

func TestRefresh_RevokedToken(t *testing.T) {
	jwtSvc := newTestJWT()
	store := newMockTokenStore()
	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ string) (*domain.User, error) {
			return &domain.User{ID: "user-1"}, nil
		},
	}
	uc := New(repo, &mockWorkspaceCreator{}, jwtSvc, store, &mockAvatarStorage{}, &mockMembershipChecker{})

	// Generate a token but do NOT save it to the store (simulates revocation)
	pair, _ := jwtSvc.GeneratePair("user-1")

	_, err := uc.Refresh(t.Context(), pair.RefreshToken)
	if !errors.Is(err, domain.ErrTokenRevoked) {
		t.Fatalf("expected ErrTokenRevoked, got: %v", err)
	}
}

func TestRefresh_RotatesTokens(t *testing.T) {
	jwtSvc := newTestJWT()
	store := newMockTokenStore()
	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ string) (*domain.User, error) {
			return &domain.User{ID: "user-1"}, nil
		},
	}
	uc := New(repo, &mockWorkspaceCreator{}, jwtSvc, store, &mockAvatarStorage{}, &mockMembershipChecker{})

	// Generate and save initial token
	pair, _ := jwtSvc.GeneratePair("user-1")
	store.Save(t.Context(), "user-1", pair.RefreshID, 7*24*time.Hour)

	// Refresh should rotate
	newPair, err := uc.Refresh(t.Context(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh error: %v", err)
	}

	// Old token should be deleted
	exists, _ := store.Exists(t.Context(), "user-1", pair.RefreshID)
	if exists {
		t.Error("old token should be deleted after rotation")
	}

	// New token should exist
	if newPair.AccessToken == "" || newPair.RefreshToken == "" {
		t.Error("expected non-empty new tokens")
	}
}

func TestLogout_DeletesToken(t *testing.T) {
	jwtSvc := newTestJWT()
	store := newMockTokenStore()
	uc := New(&mockUserRepo{}, &mockWorkspaceCreator{}, jwtSvc, store, &mockAvatarStorage{}, &mockMembershipChecker{})

	pair, _ := jwtSvc.GeneratePair("user-1")
	store.Save(t.Context(), "user-1", pair.RefreshID, 7*24*time.Hour)

	err := uc.Logout(t.Context(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("Logout error: %v", err)
	}

	exists, _ := store.Exists(t.Context(), "user-1", pair.RefreshID)
	if exists {
		t.Error("token should be deleted after logout")
	}
}

func TestOAuthLogin_NewUser(t *testing.T) {
	store := newMockTokenStore()
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, domain.ErrUserNotFound
		},
		createFn: func(_ context.Context, _ *domain.User) error {
			return nil
		},
	}
	uc := New(userRepo, &mockWorkspaceCreator{
		createFn: func(_ context.Context, _, _ string) error { return nil },
	}, newTestJWT(), store, &mockAvatarStorage{}, &mockMembershipChecker{})

	result, err := uc.OAuthLogin(t.Context(), OAuthLoginInput{
		Email:    "oauth@example.com",
		Name:     "OAuth User",
		Provider: "google",
	})
	if err != nil {
		t.Fatalf("OAuthLogin error: %v", err)
	}
	if result.User.Email != "oauth@example.com" {
		t.Errorf("email = %q, want oauth@example.com", result.User.Email)
	}
	if result.TokenPair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestOAuthLogin_ExistingUser(t *testing.T) {
	store := newMockTokenStore()
	existingUser := &domain.User{ID: "user-1", Email: "existing@example.com", Name: "Existing"}
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return existingUser, nil
		},
	}
	uc := New(userRepo, &mockWorkspaceCreator{}, newTestJWT(), store, &mockAvatarStorage{}, &mockMembershipChecker{})

	result, err := uc.OAuthLogin(t.Context(), OAuthLoginInput{
		Email:    "existing@example.com",
		Name:     "Existing",
		Provider: "github",
	})
	if err != nil {
		t.Fatalf("OAuthLogin error: %v", err)
	}
	if result.User.ID != "user-1" {
		t.Errorf("should return existing user, got ID %q", result.User.ID)
	}
}

func TestLogout_InvalidToken_NoError(t *testing.T) {
	jwtSvc := newTestJWT()
	uc := New(&mockUserRepo{}, &mockWorkspaceCreator{}, jwtSvc, newMockTokenStore(), &mockAvatarStorage{}, &mockMembershipChecker{})

	err := uc.Logout(t.Context(), "invalid.token.string")
	if err != nil {
		t.Fatalf("expected nil error for invalid token on logout, got: %v", err)
	}
}
