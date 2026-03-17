package domain_test

import (
	"testing"

	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/domain"
)

func TestRole_CanManageMembers(t *testing.T) {
	tests := []struct {
		role domain.Role
		want bool
	}{
		{domain.RoleAdmin, true},
		{domain.RolePM, true},
		{domain.RoleTechLead, false},
		{domain.RoleDeveloper, false},
		{domain.RoleObserver, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.CanManageMembers(); got != tt.want {
				t.Errorf("Role(%q).CanManageMembers() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		role domain.Role
		want bool
	}{
		{domain.RoleAdmin, true},
		{domain.RolePM, true},
		{domain.RoleTechLead, true},
		{domain.RoleDeveloper, true},
		{domain.RoleObserver, true},
		{"superadmin", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.want {
				t.Errorf("Role(%q).IsValid() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}
