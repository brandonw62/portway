package core

import "time"

// Role represents a platform-level authorization role.
type Role string

const (
	RoleAdmin        Role = "admin"
	RoleProjectOwner Role = "project-owner"
	RoleDeveloper    Role = "developer"
	RoleViewer       Role = "viewer"
)

// AllRoles returns the ordered list of roles from most to least privileged.
func AllRoles() []Role {
	return []Role{RoleAdmin, RoleProjectOwner, RoleDeveloper, RoleViewer}
}

// RoleRank returns a numeric rank for the role (higher = more privileged).
func (r Role) RoleRank() int {
	switch r {
	case RoleAdmin:
		return 100
	case RoleProjectOwner:
		return 75
	case RoleDeveloper:
		return 50
	case RoleViewer:
		return 25
	default:
		return 0
	}
}

// AtLeast returns true if this role is at least as privileged as other.
func (r Role) AtLeast(other Role) bool {
	return r.RoleRank() >= other.RoleRank()
}

// Permission represents a discrete action that can be authorized.
type Permission string

const (
	PermProjectCreate Permission = "project:create"
	PermProjectRead   Permission = "project:read"
	PermProjectUpdate Permission = "project:update"
	PermProjectDelete Permission = "project:delete"

	PermResourceProvision   Permission = "resource:provision"
	PermResourceRead        Permission = "resource:read"
	PermResourceUpdate      Permission = "resource:update"
	PermResourceDeprovision Permission = "resource:deprovision"

	PermPolicyManage Permission = "policy:manage"
	PermQuotaManage  Permission = "quota:manage"

	PermApprovalReview Permission = "approval:review"

	PermMemberInvite Permission = "member:invite"
	PermMemberRemove Permission = "member:remove"
)

// RolePermissions maps each role to its granted permissions.
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermProjectCreate, PermProjectRead, PermProjectUpdate, PermProjectDelete,
		PermResourceProvision, PermResourceRead, PermResourceUpdate, PermResourceDeprovision,
		PermPolicyManage, PermQuotaManage,
		PermApprovalReview,
		PermMemberInvite, PermMemberRemove,
	},
	RoleProjectOwner: {
		PermProjectCreate, PermProjectRead, PermProjectUpdate,
		PermResourceProvision, PermResourceRead, PermResourceUpdate, PermResourceDeprovision,
		PermApprovalReview,
		PermMemberInvite, PermMemberRemove,
	},
	RoleDeveloper: {
		PermProjectRead,
		PermResourceProvision, PermResourceRead, PermResourceUpdate,
	},
	RoleViewer: {
		PermProjectRead,
		PermResourceRead,
	},
}

// HasPermission returns true if the given role includes the specified permission.
func HasPermission(role Role, perm Permission) bool {
	for _, p := range RolePermissions[role] {
		if p == perm {
			return true
		}
	}
	return false
}

// User represents a platform user. Identity details are populated from
// SSO/OIDC and stored locally for authorization and audit purposes.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	IssuerSub string    `json:"issuer_sub"` // OIDC subject (issuer-scoped)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Membership ties a user to a project with a specific role.
type Membership struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ProjectID string    `json:"project_id"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
