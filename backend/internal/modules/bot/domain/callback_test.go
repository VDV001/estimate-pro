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

// TestConfirmCallbackPanicsOnInvalidIntent guards the constructor invariant.
// All production callers pass a literal IntentXxx constant, so an invalid
// IntentType signals a programmer error — surface it at test time rather than
// emitting "confirm:" with empty payload that the parser would happily route.
func TestConfirmCallbackPanicsOnInvalidIntent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		intent domain.IntentType
	}{
		{"empty", domain.IntentType("")},
		{"unknown literal", domain.IntentType("not_a_real_intent")},
		// IntentUnknown passes IntentType.IsValid() but is semantically
		// unconfirmable — the classifier emits it for unparseable user input,
		// and a "confirm:unknown" wire-string indicates the caller wired the
		// keyboard against the wrong intent. Reject at construction.
		{"intent_unknown", domain.IntentUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("expected panic for invalid intent %q, got none", string(tc.intent))
				}
				if msg, ok := r.(string); ok && !strings.Contains(msg, "ConfirmCallback") {
					t.Errorf("panic message missing constructor name: %q", msg)
				}
			}()
			_ = domain.ConfirmCallback(tc.intent)
		})
	}
}

// TestSelectCallbackPanicsOnUnknownKey guards the same invariant for the
// selection family. An ad-hoc key would advance the active session with an
// arbitrary state field, polluting subsequent session reads.
func TestSelectCallbackPanicsOnUnknownKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		fn   func()
	}{
		{"SelectCallback empty key", func() { _ = domain.SelectCallback(domain.CallbackKey(""), "v") }},
		{"SelectCallback unknown key", func() { _ = domain.SelectCallback(domain.CallbackKey("foo"), "v") }},
		{"SelectAction empty key", func() { _ = domain.SelectAction(domain.CallbackKey("")) }},
		{"SelectAction unknown key", func() { _ = domain.SelectAction(domain.CallbackKey("foo")) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic, got none")
				}
			}()
			tc.fn()
		})
	}
}

// TestCallbackKeyIsKnown — table-driven coverage of the whitelist gate.
func TestCallbackKeyIsKnown(t *testing.T) {
	t.Parallel()

	cases := []struct {
		key  domain.CallbackKey
		want bool
	}{
		{domain.CallbackKeyProject, true},
		{domain.CallbackKeyRole, true},
		{domain.CallbackKey(""), false},
		{domain.CallbackKey("proj_typo"), false},
	}

	for _, tc := range cases {
		t.Run(string(tc.key), func(t *testing.T) {
			t.Parallel()
			if got := tc.key.IsKnown(); got != tc.want {
				t.Errorf("CallbackKey(%q).IsKnown() = %v, want %v", string(tc.key), got, tc.want)
			}
		})
	}
}
