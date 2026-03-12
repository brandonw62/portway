package core

import "time"

// Project is the organizational unit that owns resources. Every resource
// belongs to exactly one project, and every project belongs to exactly one team.
type Project struct {
	ID          string    `json:"id"`
	TeamID      string    `json:"team_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
