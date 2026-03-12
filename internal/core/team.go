package core

import "time"

// Team is an organizational unit that owns projects and groups users together.
type Team struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TeamMemberRole defines the role a user holds within a team.
type TeamMemberRole string

const (
	TeamMemberRoleOwner  TeamMemberRole = "owner"
	TeamMemberRoleAdmin  TeamMemberRole = "admin"
	TeamMemberRoleMember TeamMemberRole = "member"
)

// TeamMember represents a user's membership in a team.
type TeamMember struct {
	TeamID    string         `json:"team_id"`
	UserID    string         `json:"user_id"`
	Role      TeamMemberRole `json:"role"`
	CreatedAt time.Time      `json:"created_at"`
}
