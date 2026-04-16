// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

var (
	ErrProjectNotFound      = errors.New("project not found")
	ErrWorkspaceNotFound    = errors.New("workspace not found")
	ErrMemberNotFound       = errors.New("member not found")
	ErrMemberAlreadyAdded   = errors.New("member already added to project")
	ErrInsufficientRole     = errors.New("insufficient role for this action")
	ErrInvalidWorkspaceName = errors.New("workspace name must be 1..255 characters")
	ErrMissingOwner         = errors.New("workspace owner is required")
	ErrWorkspaceForbidden   = errors.New("only workspace owner can modify")
	ErrInvalidProjectName   = errors.New("project name must be 1..255 characters")
	ErrMissingWorkspace     = errors.New("project workspace is required")
	ErrMissingCreator       = errors.New("project creator is required")
	ErrMissingProject       = errors.New("member project is required")
	ErrMissingUser          = errors.New("member user is required")
	ErrInvalidRole          = errors.New("member role is invalid")
)
