// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "fmt"

// Callback protocol for Telegram inline-keyboard buttons.
//
// Callback data follows the convention "action:payload" (see ADR-011).
// Producer side (keyboard builders) calls the constructors below; parser
// side (BotUsecase.ProcessCallback) compares the parsed action against the
// action constants. Adding a new action prefix or selection key requires
// touching both sides — keep the contract rooted in this file.
const (
	CallbackActionCancel  = "cancel"
	CallbackActionConfirm = "confirm"

	CallbackPrefixSelect = "sel_"
)

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
	return CallbackActionCancel + ":"
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
	return CallbackActionConfirm + ":" + string(intent)
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
func SelectAction(key CallbackKey) string {
	if !key.IsKnown() {
		panic(fmt.Sprintf("domain.SelectAction: unknown CallbackKey %q", string(key)))
	}
	return CallbackPrefixSelect + string(key)
}
