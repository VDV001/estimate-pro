// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

// TestCallbackConstructors locks the wire format produced by the helpers —
// changing any output here breaks the parser side of ProcessCallback and any
// inline keyboards still in chat history. Update parseCallbackData in lockstep.
func TestCallbackConstructors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"cancel", domain.CancelCallback(), "cancel:"},
		{"confirm create_project", domain.ConfirmCallback(domain.IntentCreateProject), "confirm:create_project"},
		{"confirm update_project", domain.ConfirmCallback(domain.IntentUpdateProject), "confirm:update_project"},
		{"confirm remove_member", domain.ConfirmCallback(domain.IntentRemoveMember), "confirm:remove_member"},
		{"select role", domain.SelectCallback(domain.CallbackKeyRole, "developer"), "sel_role:developer"},
		{"select project", domain.SelectCallback(domain.CallbackKeyProject, "abc-123"), "sel_proj:abc-123"},
		{"select action role", domain.SelectAction(domain.CallbackKeyRole), "sel_role"},
		{"select action proj", domain.SelectAction(domain.CallbackKeyProject), "sel_proj"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Errorf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}

// TestSelectActionMatchesSelectCallbackPrefix asserts the parser-side
// invariant: SelectAction(key) is always a prefix of SelectCallback(key, *).
// If this ever fails, ProcessCallback's strings.HasPrefix check would miss
// keyboards built by SelectCallback.
func TestSelectActionMatchesSelectCallbackPrefix(t *testing.T) {
	t.Parallel()

	keys := []string{domain.CallbackKeyRole, domain.CallbackKeyProject}
	for _, k := range keys {
		full := domain.SelectCallback(k, "anything")
		action := domain.SelectAction(k)
		if !strings.HasPrefix(full, domain.CallbackPrefixSelect) {
			t.Errorf("SelectCallback(%q,*) = %q does not start with %q", k, full, domain.CallbackPrefixSelect)
		}
		if !strings.HasPrefix(full, action+":") {
			t.Errorf("SelectAction(%q) = %q is not a prefix of SelectCallback(%q,*) = %q", k, action, k, full)
		}
	}
}
