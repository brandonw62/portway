package core

import "time"

// AuditAction categorizes what happened.
type AuditAction string

const (
	AuditResourceProvision   AuditAction = "resource.provision"
	AuditResourceDeprovision AuditAction = "resource.deprovision"
	AuditResourceUpdate      AuditAction = "resource.update"
	AuditPolicyEvaluated     AuditAction = "policy.evaluated"
	AuditPolicyDenied        AuditAction = "policy.denied"
	AuditApprovalRequested   AuditAction = "approval.requested"
	AuditApprovalReviewed    AuditAction = "approval.reviewed"
	AuditQuotaExceeded       AuditAction = "quota.exceeded"
	AuditMemberAdded         AuditAction = "member.added"
	AuditMemberRemoved       AuditAction = "member.removed"
)

// AuditEntry records a single auditable event in the platform.
type AuditEntry struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actor_id"`              // user who performed the action
	ProjectID  string         `json:"project_id,omitempty"`
	Action     AuditAction    `json:"action"`
	TargetType string         `json:"target_type,omitempty"` // e.g. "resource", "approval", "policy"
	TargetID   string         `json:"target_id,omitempty"`
	Detail     map[string]any `json:"detail,omitempty"`      // action-specific structured data
	Allowed    bool           `json:"allowed"`               // whether the action was permitted
	CreatedAt  time.Time      `json:"created_at"`
}
