package domain

import "errors"

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrWorkspaceNotFound  = errors.New("workspace not found")
	ErrMemberNotFound     = errors.New("member not found")
	ErrMemberAlreadyAdded = errors.New("member already added to project")
	ErrInsufficientRole   = errors.New("insufficient role for this action")
)
