// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

// Callback protocol for Telegram inline-keyboard buttons.
//
// Callback data follows the convention "action:payload" (see ADR-011).
// Producer side (keyboard builders) uses the constructors below; parser side
// (BotUsecase.ProcessCallback) compares the parsed action against the action
// constants. Adding a new prefix or key requires touching both sides — keep
// them rooted in this file so the contract is in one place.
const (
	CallbackActionCancel  = "cancel"
	CallbackActionConfirm = "confirm"

	CallbackPrefixSelect = "sel_"

	CallbackKeyProject = "proj"
	CallbackKeyRole    = "role"
)

// CancelCallback returns the callback_data string for a "cancel" button.
func CancelCallback() string {
	return CallbackActionCancel + ":"
}

// ConfirmCallback returns the callback_data string for a "confirm" button
// scoped to a specific intent (e.g. "confirm:create_project").
func ConfirmCallback(intent IntentType) string {
	return CallbackActionConfirm + ":" + string(intent)
}

// SelectCallback returns the callback_data string for a selection button
// (e.g. "sel_role:developer", "sel_proj:<project-id>").
func SelectCallback(key, value string) string {
	return CallbackPrefixSelect + key + ":" + value
}

// SelectAction returns the parsed action portion produced by SelectCallback —
// useful for parser-side comparisons (e.g. action == SelectAction(CallbackKeyProject)).
func SelectAction(key string) string {
	return CallbackPrefixSelect + key
}
