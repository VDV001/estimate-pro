// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"fmt"
	"strings"
)

// Callback protocol for Telegram inline-keyboard buttons.
//
// Callback data follows the convention "action:payload" (see ADR-011 / ADR-013).
// Producer side (keyboard builders) calls the constructors below; parser
// side (BotUsecase.ProcessCallback) routes via the typed CallbackAction
// returned from ParseCallback. Adding a new action prefix or selection key
// requires touching both sides — keep the contract rooted in this file.

// CallbackPrefixSelect is the literal prefix shared by all selection actions
// (e.g. "sel_proj", "sel_role"). Exported because the wire format is part of
// the cross-version backward-compat contract.
const CallbackPrefixSelect = "sel_"

// CallbackAction is the parsed action portion of a callback_data string —
// the substring before the first colon. Cancel and Confirm have fixed wire
// values; select actions take the dynamic form CallbackPrefixSelect + key.
type CallbackAction string

const (
	CallbackActionCancel  CallbackAction = "cancel"
	CallbackActionConfirm CallbackAction = "confirm"
)

// IsCancel reports whether the action is the cancel sentinel.
func (a CallbackAction) IsCancel() bool { return a == CallbackActionCancel }

// IsConfirm reports whether the action is the confirm sentinel.
func (a CallbackAction) IsConfirm() bool { return a == CallbackActionConfirm }

// IsSelect reports whether the action is a selection action (sel_<key>).
// Returns true even for unknown keys — use SelectKey().IsKnown() to gate.
func (a CallbackAction) IsSelect() bool {
	return strings.HasPrefix(string(a), CallbackPrefixSelect)
}

// SelectKey returns the CallbackKey embedded in a select action. Returns ""
// for non-select actions. The returned key may be unknown — caller is
// expected to call IsKnown() before treating it as authoritative.
func (a CallbackAction) SelectKey() CallbackKey {
	if !a.IsSelect() {
		return ""
	}
	return CallbackKey(strings.TrimPrefix(string(a), CallbackPrefixSelect))
}

// CallbackKey is a whitelisted selection-key for SelectCallback / SelectAction.
// New keys must be declared here and accepted by IsKnown — the constructors
// panic on unknown keys to surface programmer errors at test time rather than
// emitting a malformed wire-string that the parser would silently ignore.
type CallbackKey string

const (
	CallbackKeyProject CallbackKey = "proj"
	CallbackKeyRole    CallbackKey = "role"
)

// IsKnown reports whether the key is one of the declared CallbackKey constants.
func (k CallbackKey) IsKnown() bool {
	switch k {
	case CallbackKeyProject, CallbackKeyRole:
		return true
	default:
		return false
	}
}

// CancelCallback returns the callback_data string for a "cancel" button.
func CancelCallback() string {
	return string(CallbackActionCancel) + ":"
}

// ConfirmCallback returns the callback_data string for a "confirm" button
// scoped to a specific intent (e.g. "confirm:create_project").
//
// Panics if intent is not a known IntentType, or is IntentUnknown — the
// classifier emits IntentUnknown for unparseable input, and there is no
// session action to confirm for it. Programmer-error idiom mirrors
// regexp.MustCompile / template.Must.
func ConfirmCallback(intent IntentType) string {
	if !intent.IsValid() || intent == IntentUnknown {
		panic(fmt.Sprintf("domain.ConfirmCallback: invalid IntentType %q", string(intent)))
	}
	return string(CallbackActionConfirm) + ":" + string(intent)
}

// SelectCallback returns the callback_data string for a selection button
// (e.g. "sel_role:developer", "sel_proj:<project-id>").
//
// Panics if key is not a declared CallbackKey constant — protects the wire
// format from accidental keys that ProcessCallback would advance the session
// with, polluting session state with arbitrary keys.
func SelectCallback(key CallbackKey, value string) string {
	if !key.IsKnown() {
		panic(fmt.Sprintf("domain.SelectCallback: unknown CallbackKey %q", string(key)))
	}
	return CallbackPrefixSelect + string(key) + ":" + value
}

// SelectAction returns the parsed action portion produced by SelectCallback —
// useful for parser-side comparisons (e.g. action == SelectAction(CallbackKeyProject)).
//
// Panics on unknown CallbackKey for the same reason as SelectCallback.
func SelectAction(key CallbackKey) CallbackAction {
	if !key.IsKnown() {
		panic(fmt.Sprintf("domain.SelectAction: unknown CallbackKey %q", string(key)))
	}
	return CallbackAction(CallbackPrefixSelect + string(key))
}

// ParseCallback splits Telegram callback_data into its typed action and
// raw payload using the "action:payload" convention (ADR-011).
//
// Backward-compatible: a legacy value without a colon yields the whole
// string as action and empty payload — old inline keyboards still in chat
// history continue to parse.
func ParseCallback(data string) (action CallbackAction, payload string) {
	parts := strings.SplitN(data, ":", 2)
	action = CallbackAction(parts[0])
	if len(parts) == 2 {
		payload = parts[1]
	}
	return
}
