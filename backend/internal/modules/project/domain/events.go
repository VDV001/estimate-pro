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
