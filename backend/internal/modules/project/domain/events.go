// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

type MemberInvited struct {
	ProjectID string
	UserID    string
	InvitedBy string
	Role      Role
}

type ProjectStatusChanged struct {
	ProjectID string
	OldStatus ProjectStatus
	NewStatus ProjectStatus
	ChangedBy string
}
