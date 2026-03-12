package core

import "testing"

func TestEvaluatePolicies_AllowByDefault(t *testing.T) {
	result := EvaluatePolicies(nil, ProvisionRequest{ResourceType: "database"})
	if !result.Allowed {
		t.Fatal("expected allow when no policies defined")
	}
}

func TestEvaluatePolicies_DenyOverridesApproval(t *testing.T) {
	policies := []Policy{
		{
			ID: "p1", Enabled: true, Scope: PolicyScopeGlobal,
			Rules: []PolicyRule{
				{ResourceType: "database", Attribute: "environment", Operator: OpEquals, Value: "production", Effect: PolicyRequireApproval},
			},
		},
		{
			ID: "p2", Enabled: true, Scope: PolicyScopeGlobal,
			Rules: []PolicyRule{
				{ResourceType: "database", Attribute: "instance_size", Operator: OpEquals, Value: "r5.4xlarge", Effect: PolicyDeny, Description: "instance too large"},
			},
		},
	}
	req := ProvisionRequest{
		ResourceType: "database",
		Environment:  "production",
		Attributes:   map[string]string{"instance_size": "r5.4xlarge"},
	}
	result := EvaluatePolicies(policies, req)
	if result.Allowed {
		t.Fatal("expected deny")
	}
	if result.Effect != PolicyDeny {
		t.Fatalf("expected effect deny, got %s", result.Effect)
	}
	if len(result.DenyReasons) != 1 {
		t.Fatalf("expected 1 deny reason, got %d", len(result.DenyReasons))
	}
}

func TestEvaluatePolicies_RequireApproval(t *testing.T) {
	policies := []Policy{
		{
			ID: "p1", Enabled: true, Scope: PolicyScopeGlobal,
			Rules: []PolicyRule{
				{ResourceType: "*", Attribute: "environment", Operator: OpEquals, Value: "production", Effect: PolicyRequireApproval, Description: "prod needs approval"},
			},
		},
	}
	req := ProvisionRequest{ResourceType: "compute", Environment: "production"}
	result := EvaluatePolicies(policies, req)
	if result.Allowed {
		t.Fatal("expected not allowed")
	}
	if result.Effect != PolicyRequireApproval {
		t.Fatalf("expected require_approval, got %s", result.Effect)
	}
}

func TestEvaluatePolicies_SkipsDisabledAndWrongProject(t *testing.T) {
	policies := []Policy{
		{ID: "disabled", Enabled: false, Scope: PolicyScopeGlobal, Rules: []PolicyRule{
			{ResourceType: "*", Attribute: "environment", Operator: OpEquals, Value: "production", Effect: PolicyDeny},
		}},
		{ID: "other-proj", Enabled: true, Scope: PolicyScopeProject, ProjectID: "proj-other", Rules: []PolicyRule{
			{ResourceType: "*", Attribute: "environment", Operator: OpEquals, Value: "production", Effect: PolicyDeny},
		}},
	}
	req := ProvisionRequest{ProjectID: "proj-mine", ResourceType: "database", Environment: "production"}
	result := EvaluatePolicies(policies, req)
	if !result.Allowed {
		t.Fatal("expected allow — both policies should be skipped")
	}
}

func TestEvaluatePolicies_InOperator(t *testing.T) {
	policies := []Policy{
		{ID: "p1", Enabled: true, Scope: PolicyScopeGlobal, Rules: []PolicyRule{
			{ResourceType: "compute", Attribute: "region", Operator: OpNotIn, Value: "us-east-1,us-west-2,eu-west-1", Effect: PolicyDeny, Description: "region not allowed"},
		}},
	}
	// Allowed region
	req := ProvisionRequest{ResourceType: "compute", Attributes: map[string]string{"region": "us-east-1"}}
	result := EvaluatePolicies(policies, req)
	if !result.Allowed {
		t.Fatal("us-east-1 should be allowed")
	}

	// Denied region
	req.Attributes["region"] = "ap-southeast-1"
	result = EvaluatePolicies(policies, req)
	if result.Allowed {
		t.Fatal("ap-southeast-1 should be denied")
	}
}
