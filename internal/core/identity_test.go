package core

import "testing"

func TestRoleAtLeast(t *testing.T) {
	tests := []struct {
		role   Role
		target Role
		want   bool
	}{
		{RoleAdmin, RoleViewer, true},
		{RoleAdmin, RoleAdmin, true},
		{RoleDeveloper, RoleProjectOwner, false},
		{RoleViewer, RoleDeveloper, false},
		{RoleProjectOwner, RoleDeveloper, true},
	}
	for _, tt := range tests {
		if got := tt.role.AtLeast(tt.target); got != tt.want {
			t.Errorf("%s.AtLeast(%s) = %v, want %v", tt.role, tt.target, got, tt.want)
		}
	}
}

func TestHasPermission(t *testing.T) {
	if !HasPermission(RoleAdmin, PermPolicyManage) {
		t.Error("admin should have policy:manage")
	}
	if HasPermission(RoleDeveloper, PermPolicyManage) {
		t.Error("developer should not have policy:manage")
	}
	if !HasPermission(RoleDeveloper, PermResourceProvision) {
		t.Error("developer should have resource:provision")
	}
	if HasPermission(RoleViewer, PermResourceProvision) {
		t.Error("viewer should not have resource:provision")
	}
}
