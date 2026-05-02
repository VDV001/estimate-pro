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
		{"select action role", string(domain.SelectAction(domain.CallbackKeyRole)), "sel_role"},
		{"select action proj", string(domain.SelectAction(domain.CallbackKeyProject)), "sel_proj"},
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

// TestCallbackActionPredicates — predicate methods routed on the typed
// action. Each row exercises exactly one of IsCancel / IsConfirm / IsSelect
// to lock the disjoint partitioning of action space.
func TestCallbackActionPredicates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		action    domain.CallbackAction
		isCancel  bool
		isConfirm bool
		isSelect  bool
	}{
		{"cancel", domain.CallbackActionCancel, true, false, false},
		{"confirm", domain.CallbackActionConfirm, false, true, false},
		{"select role", domain.SelectAction(domain.CallbackKeyRole), false, false, true},
		{"select proj", domain.SelectAction(domain.CallbackKeyProject), false, false, true},
		{"select unknown key still IsSelect", domain.CallbackAction("sel_unknown"), false, false, true},
		{"empty action", domain.CallbackAction(""), false, false, false},
		{"foreign action", domain.CallbackAction("nuke"), false, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.action.IsCancel(); got != tc.isCancel {
				t.Errorf("IsCancel() = %v, want %v", got, tc.isCancel)
			}
			if got := tc.action.IsConfirm(); got != tc.isConfirm {
				t.Errorf("IsConfirm() = %v, want %v", got, tc.isConfirm)
			}
			if got := tc.action.IsSelect(); got != tc.isSelect {
				t.Errorf("IsSelect() = %v, want %v", got, tc.isSelect)
			}
		})
	}
}

// TestCallbackActionSelectKey — SelectKey extracts the typed key portion.
// Returns "" for non-select actions and an unknown CallbackKey for select
// actions whose key is not in the whitelist (caller must check IsKnown).
func TestCallbackActionSelectKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		action  domain.CallbackAction
		want    domain.CallbackKey
		wantOk  bool
	}{
		{"select role", domain.SelectAction(domain.CallbackKeyRole), domain.CallbackKeyRole, true},
		{"select proj", domain.SelectAction(domain.CallbackKeyProject), domain.CallbackKeyProject, true},
		{"select unknown key", domain.CallbackAction("sel_unknown"), domain.CallbackKey("unknown"), false},
		{"cancel action", domain.CallbackActionCancel, domain.CallbackKey(""), false},
		{"confirm action", domain.CallbackActionConfirm, domain.CallbackKey(""), false},
		{"empty action", domain.CallbackAction(""), domain.CallbackKey(""), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.action.SelectKey()
			if got != tc.want {
				t.Errorf("SelectKey() = %q, want %q", string(got), string(tc.want))
			}
			if ok := got.IsKnown(); ok != tc.wantOk {
				t.Errorf("SelectKey().IsKnown() = %v, want %v", ok, tc.wantOk)
			}
		})
	}
}

// TestParseCallback covers the wire-format split for canonical, malformed,
// and legacy callback_data values. Legacy "cancel" without colon must still
// parse to action=cancel for chats that have keyboards from before #20.
func TestParseCallback(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		in          string
		wantAction  domain.CallbackAction
		wantPayload string
	}{
		{"cancel without colon (legacy)", "cancel", domain.CallbackActionCancel, ""},
		{"cancel canonical", "cancel:", domain.CallbackActionCancel, ""},
		{"confirm with intent payload", "confirm:create_project", domain.CallbackActionConfirm, "create_project"},
		{"select project with id", "sel_proj:abc-123", domain.SelectAction(domain.CallbackKeyProject), "abc-123"},
		{"select role with developer", "sel_role:developer", domain.SelectAction(domain.CallbackKeyRole), "developer"},
		{"empty input", "", domain.CallbackAction(""), ""},
		{"action only no colon", "foo", domain.CallbackAction("foo"), ""},
		{"colon at end empty payload", "foo:", domain.CallbackAction("foo"), ""},
		{"multiple colons preserve payload after first", "a:b:c", domain.CallbackAction("a"), "b:c"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotAction, gotPayload := domain.ParseCallback(tc.in)
			if gotAction != tc.wantAction {
				t.Errorf("action = %q, want %q", string(gotAction), string(tc.wantAction))
			}
			if gotPayload != tc.wantPayload {
				t.Errorf("payload = %q, want %q", gotPayload, tc.wantPayload)
			}
		})
	}
}

// TestParseCallbackRoundTrip — wire-format identity: callback_data emitted
// by the producer constructors round-trips through ParseCallback to the
// expected typed action + payload. Locks the producer/parser contract end-to-end.
func TestParseCallbackRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		emit        string
		wantAction  domain.CallbackAction
		wantPayload string
	}{
		{"CancelCallback", domain.CancelCallback(), domain.CallbackActionCancel, ""},
		{"ConfirmCallback create", domain.ConfirmCallback(domain.IntentCreateProject), domain.CallbackActionConfirm, "create_project"},
		{"SelectCallback role", domain.SelectCallback(domain.CallbackKeyRole, "developer"), domain.SelectAction(domain.CallbackKeyRole), "developer"},
		{"SelectCallback proj uuid-shape", domain.SelectCallback(domain.CallbackKeyProject, "00000000-0000-0000-0000-000000000001"), domain.SelectAction(domain.CallbackKeyProject), "00000000-0000-0000-0000-000000000001"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotAction, gotPayload := domain.ParseCallback(tc.emit)
			if gotAction != tc.wantAction {
				t.Errorf("action = %q, want %q (emit=%q)", string(gotAction), string(tc.wantAction), tc.emit)
			}
			if gotPayload != tc.wantPayload {
				t.Errorf("payload = %q, want %q (emit=%q)", gotPayload, tc.wantPayload, tc.emit)
			}
		})
	}
}
