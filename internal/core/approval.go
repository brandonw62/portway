package core

import "time"

// ApprovalStatus tracks the lifecycle of an approval request.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalDenied   ApprovalStatus = "denied"
	ApprovalExpired  ApprovalStatus = "expired"
)

// ApprovalRequest is created when a policy evaluation returns PolicyRequireApproval.
type ApprovalRequest struct {
	ID               string         `json:"id"`
	ProjectID        string         `json:"project_id"`
	RequestedBy      string         `json:"requested_by"`       // user ID
	ResourceType     string         `json:"resource_type"`
	RequestPayload   map[string]any `json:"request_payload"`    // the original provision request details
	Reasons          []string       `json:"reasons"`            // why approval is required
	MatchedPolicies  []string       `json:"matched_policies"`
	Status           ApprovalStatus `json:"status"`
	ReviewedBy       string         `json:"reviewed_by,omitempty"`
	ReviewComment    string         `json:"review_comment,omitempty"`
	ReviewedAt       *time.Time     `json:"reviewed_at,omitempty"`
	ExpiresAt        time.Time      `json:"expires_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// IsResolved returns true if the approval has been reviewed or expired.
func (a ApprovalRequest) IsResolved() bool {
	return a.Status != ApprovalPending
}
