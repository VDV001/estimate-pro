// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already taken")
	ErrTokenRevoked       = errors.New("refresh token revoked")
	ErrResetTokenNotFound = errors.New("reset token not found or expired")
	ErrNoPassword         = errors.New("account uses OAuth login, no password to reset")
)
