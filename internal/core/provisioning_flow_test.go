package core

import "testing"

// TestProvisioningFlowPolicyDeniesLargeInstances verifies that a policy denying
// large instances correctly blocks provisioning.
func TestProvisioningFlowPolicyDeniesLargeInstances(t *testing.T) {
	policies := []Policy{
		{
			ID: "size-limit", Name: "Instance Size Limit", Enabled: true, Scope: PolicyScopeGlobal,
			Rules: []PolicyRule{
				{ResourceType: "database", Attribute: "instance_size", Operator: OpIn, Value: "r5.4xlarge,r5.8xlarge,r5.16xlarge", Effect: PolicyDeny, Description: "oversized instances not allowed"},
			},
		},
	}

	// Small instance should be allowed.
	req := ProvisionRequest{
		ResourceType: "database",
		Attributes:   map[string]string{"instance_size": "t3.medium"},
	}
	result := EvaluatePolicies(policies, req)
	if !result.Allowed {
		t.Fatal("t3.medium should be allowed")
	}

	// Large instance should be denied.
	req.Attributes["instance_size"] = "r5.4xlarge"
	result = EvaluatePolicies(policies, req)
	if result.Allowed {
		t.Fatal("r5.4xlarge should be denied")
	}
	if result.Effect != PolicyDeny {
		t.Fatalf("expected deny effect, got %s", result.Effect)
	}
	if len(result.DenyReasons) != 1 || result.DenyReasons[0] != "oversized instances not allowed" {
		t.Fatalf("unexpected deny reasons: %v", result.DenyReasons)
	}
}

// TestProvisioningFlowProductionRequiresApproval verifies that production
// resources trigger an approval workflow.
func TestProvisioningFlowProductionRequiresApproval(t *testing.T) {
	policies := []Policy{
		{
			ID: "prod-gate", Name: "Production Gate", Enabled: true, Scope: PolicyScopeGlobal,
			Rules: []PolicyRule{
				{ResourceType: "*", Attribute: "environment", Operator: OpEquals, Value: "production", Effect: PolicyRequireApproval, Description: "production resources require approval"},
			},
		},
	}

	// Dev environment should pass through.
	req := ProvisionRequest{ResourceType: "database", Environment: "development"}
	result := EvaluatePolicies(policies, req)
	if !result.Allowed {
		t.Fatal("development should be allowed")
	}

	// Production should require approval.
	req.Environment = "production"
	result = EvaluatePolicies(policies, req)
	if result.Allowed {
		t.Fatal("production should not be directly allowed")
	}
	if result.Effect != PolicyRequireApproval {
		t.Fatalf("expected require_approval, got %s", result.Effect)
	}
	if len(result.ApprovalReasons) != 1 {
		t.Fatalf("expected 1 approval reason, got %d", len(result.ApprovalReasons))
	}
}

// TestProvisioningFlowDenyOverridesApprovalForProduction verifies deny takes
// precedence even when another policy says require_approval.
func TestProvisioningFlowDenyOverridesApprovalForProduction(t *testing.T) {
	policies := []Policy{
		{
			ID: "prod-gate", Name: "Production Gate", Enabled: true, Scope: PolicyScopeGlobal,
			Rules: []PolicyRule{
				{ResourceType: "*", Attribute: "environment", Operator: OpEquals, Value: "production", Effect: PolicyRequireApproval},
			},
		},
		{
			ID: "region-lock", Name: "Region Lock", Enabled: true, Scope: PolicyScopeGlobal,
			Rules: []PolicyRule{
				{ResourceType: "*", Attribute: "region", Operator: OpNotIn, Value: "us-east-1,us-west-2", Effect: PolicyDeny, Description: "region not allowed"},
			},
		},
	}

	req := ProvisionRequest{
		ResourceType: "database",
		Environment:  "production",
		Attributes:   map[string]string{"region": "ap-southeast-1"},
	}
	result := EvaluatePolicies(policies, req)
	if result.Allowed {
		t.Fatal("should be denied")
	}
	if result.Effect != PolicyDeny {
		t.Fatalf("deny should override require_approval, got %s", result.Effect)
	}
	if len(result.DenyReasons) == 0 {
		t.Fatal("expected deny reasons")
	}
	if len(result.ApprovalReasons) == 0 {
		t.Fatal("approval reasons should still be captured even when denied")
	}
}

// TestProvisioningFlowProjectScopedPoliciesOverride verifies that project-scoped
// policies only apply to their project.
func TestProvisioningFlowProjectScopedPoliciesOverride(t *testing.T) {
	policies := []Policy{
		{
			ID: "proj-strict", Name: "Project Strict Mode", Enabled: true,
			Scope: PolicyScopeProject, ProjectID: "proj-secure",
			Rules: []PolicyRule{
				{ResourceType: "database", Attribute: "encryption", Operator: OpNotEquals, Value: "true", Effect: PolicyDeny, Description: "databases must be encrypted"},
			},
		},
	}

	// Different project — policy doesn't apply.
	req := ProvisionRequest{
		ProjectID:    "proj-other",
		ResourceType: "database",
		Attributes:   map[string]string{"encryption": "false"},
	}
	result := EvaluatePolicies(policies, req)
	if !result.Allowed {
		t.Fatal("policy should not apply to proj-other")
	}

	// Matching project, unencrypted — denied.
	req.ProjectID = "proj-secure"
	result = EvaluatePolicies(policies, req)
	if result.Allowed {
		t.Fatal("unencrypted database should be denied in proj-secure")
	}

	// Matching project, encrypted — allowed.
	req.Attributes["encryption"] = "true"
	result = EvaluatePolicies(policies, req)
	if !result.Allowed {
		t.Fatal("encrypted database should be allowed in proj-secure")
	}
}

// TestProvisioningFlowMultiplePoliciesAllMatch verifies that all matching
// policies are evaluated and their effects combined.
func TestProvisioningFlowMultiplePoliciesAllMatch(t *testing.T) {
	policies := []Policy{
		{
			ID: "p1", Name: "Rule 1", Enabled: true, Scope: PolicyScopeGlobal,
			Rules: []PolicyRule{
				{ResourceType: "*", Attribute: "environment", Operator: OpEquals, Value: "production", Effect: PolicyRequireApproval, Description: "prod needs approval"},
			},
		},
		{
			ID: "p2", Name: "Rule 2", Enabled: true, Scope: PolicyScopeGlobal,
			Rules: []PolicyRule{
				{ResourceType: "database", Attribute: "engine", Operator: OpNotIn, Value: "postgres,mysql", Effect: PolicyDeny, Description: "unsupported engine"},
			},
		},
	}

	// Production postgres — require_approval only.
	req := ProvisionRequest{
		ResourceType: "database",
		Environment:  "production",
		Attributes:   map[string]string{"engine": "postgres"},
	}
	result := EvaluatePolicies(policies, req)
	if result.Effect != PolicyRequireApproval {
		t.Fatalf("expected require_approval, got %s", result.Effect)
	}
	if len(result.MatchedPolicies) != 1 || result.MatchedPolicies[0] != "p1" {
		t.Fatalf("expected only p1 to match, got %v", result.MatchedPolicies)
	}

	// Production mongodb — deny (both policies match, deny wins).
	req.Attributes["engine"] = "mongodb"
	result = EvaluatePolicies(policies, req)
	if result.Effect != PolicyDeny {
		t.Fatalf("expected deny, got %s", result.Effect)
	}
	if len(result.MatchedPolicies) != 2 {
		t.Fatalf("expected 2 matched policies, got %d", len(result.MatchedPolicies))
	}
}

// TestProvisioningFlowQuotaEnforcement verifies quota checking at different usage levels.
func TestProvisioningFlowQuotaEnforcement(t *testing.T) {
	quotas := []Quota{
		{ID: "global-db", ResourceType: "database", Limit: 10},
		{ID: "proj-db", ProjectID: "proj-1", ResourceType: "database", Limit: 3},
		{ID: "proj-all", ProjectID: "proj-1", ResourceType: "*", Limit: 20},
	}

	tests := []struct {
		name         string
		projectID    string
		resourceType string
		current      int
		wantExceeded bool
	}{
		{"within project db limit", "proj-1", "database", 2, false},
		{"at project db limit", "proj-1", "database", 3, true},
		{"over project db limit", "proj-1", "database", 5, true},
		{"within global db limit", "proj-2", "database", 9, false},
		{"at global db limit", "proj-2", "database", 10, true},
		{"proj wildcard within limit", "proj-1", "cache", 19, false},
		{"proj wildcard at limit", "proj-1", "cache", 20, true},
		{"no quota for type", "proj-2", "cache", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := CheckQuota(quotas, tt.projectID, tt.resourceType, tt.current)
			if tt.wantExceeded {
				if usage == nil || !usage.Exceeded() {
					t.Fatalf("expected exceeded for %s at %d", tt.resourceType, tt.current)
				}
			} else {
				if usage != nil && usage.Exceeded() {
					t.Fatalf("expected not exceeded for %s at %d, but got exceeded (limit=%d)", tt.resourceType, tt.current, usage.Limit)
				}
			}
		})
	}
}

// TestProvisioningFlowPolicyWithNoAttributes verifies rules that reference
// attributes not present in the request are silently skipped.
func TestProvisioningFlowPolicyWithNoAttributes(t *testing.T) {
	policies := []Policy{
		{
			ID: "p1", Name: "Optional Check", Enabled: true, Scope: PolicyScopeGlobal,
			Rules: []PolicyRule{
				{ResourceType: "database", Attribute: "cost_center", Operator: OpEquals, Value: "", Effect: PolicyDeny, Description: "cost center required"},
			},
		},
	}

	// Request without cost_center attribute — rule should not match.
	req := ProvisionRequest{
		ResourceType: "database",
		Attributes:   map[string]string{"engine": "postgres"},
	}
	result := EvaluatePolicies(policies, req)
	if !result.Allowed {
		t.Fatal("should be allowed when attribute is missing from request")
	}
}
