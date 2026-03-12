package core

import (
	"fmt"
	"time"
)

// PolicyEffect is the outcome when a policy rule matches.
type PolicyEffect string

const (
	PolicyAllow           PolicyEffect = "allow"
	PolicyDeny            PolicyEffect = "deny"
	PolicyRequireApproval PolicyEffect = "require_approval"
)

// PolicyScope determines what a policy applies to.
type PolicyScope string

const (
	PolicyScopeGlobal  PolicyScope = "global"
	PolicyScopeProject PolicyScope = "project"
)

// Policy defines a set of rules that govern resource provisioning.
type Policy struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Scope       PolicyScope `json:"scope"`
	ProjectID   string      `json:"project_id,omitempty"` // set when Scope == PolicyScopeProject
	Rules       []PolicyRule `json:"rules"`
	Enabled     bool        `json:"enabled"`
	CreatedBy   string      `json:"created_by"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// PolicyRule is one constraint within a policy.
type PolicyRule struct {
	ID           string            `json:"id"`
	Description  string            `json:"description,omitempty"`
	ResourceType string            `json:"resource_type"` // e.g. "database", "compute", "*"
	Attribute    string            `json:"attribute"`     // e.g. "instance_size", "region", "environment"
	Operator     RuleOperator      `json:"operator"`
	Value        string            `json:"value"`         // compared against the resource request
	Effect       PolicyEffect      `json:"effect"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// RuleOperator defines how a policy rule's value is compared.
type RuleOperator string

const (
	OpEquals      RuleOperator = "eq"
	OpNotEquals   RuleOperator = "neq"
	OpIn          RuleOperator = "in"  // comma-separated list in Value
	OpNotIn       RuleOperator = "nin"
	OpLessThan    RuleOperator = "lt"
	OpGreaterThan RuleOperator = "gt"
	OpLessEq      RuleOperator = "lte"
	OpGreaterEq   RuleOperator = "gte"
)

// ProvisionRequest is the input to policy evaluation — represents what a
// user is trying to provision.
type ProvisionRequest struct {
	UserID       string            `json:"user_id"`
	ProjectID    string            `json:"project_id"`
	ResourceType string            `json:"resource_type"`
	Environment  string            `json:"environment"` // e.g. "production", "staging", "development"
	Attributes   map[string]string `json:"attributes"`  // key-value pairs describing the resource
}

// EvalResult is the outcome of evaluating policies against a provision request.
type EvalResult struct {
	Allowed         bool         `json:"allowed"`
	Effect          PolicyEffect `json:"effect"`
	DenyReasons     []string     `json:"deny_reasons,omitempty"`
	ApprovalReasons []string     `json:"approval_reasons,omitempty"`
	MatchedPolicies []string     `json:"matched_policies"` // IDs of policies that matched
}

// EvaluatePolicies runs the given policies against a provision request.
// Deny rules take priority; then require-approval; default is allow.
func EvaluatePolicies(policies []Policy, req ProvisionRequest) EvalResult {
	result := EvalResult{Allowed: true, Effect: PolicyAllow}

	for _, p := range policies {
		if !p.Enabled {
			continue
		}
		if p.Scope == PolicyScopeProject && p.ProjectID != req.ProjectID {
			continue
		}
		for _, rule := range p.Rules {
			if !ruleMatchesResource(rule, req) {
				continue
			}
			val, ok := resolveAttribute(rule.Attribute, req)
			if !ok {
				continue
			}
			if !evaluateCondition(rule.Operator, val, rule.Value) {
				continue
			}
			result.MatchedPolicies = append(result.MatchedPolicies, p.ID)
			switch rule.Effect {
			case PolicyDeny:
				result.Allowed = false
				result.Effect = PolicyDeny
				reason := rule.Description
				if reason == "" {
					reason = fmt.Sprintf("policy %q denied: %s %s %s", p.Name, rule.Attribute, rule.Operator, rule.Value)
				}
				result.DenyReasons = append(result.DenyReasons, reason)
			case PolicyRequireApproval:
				if result.Effect != PolicyDeny {
					result.Effect = PolicyRequireApproval
					result.Allowed = false
				}
				reason := rule.Description
				if reason == "" {
					reason = fmt.Sprintf("policy %q requires approval: %s %s %s", p.Name, rule.Attribute, rule.Operator, rule.Value)
				}
				result.ApprovalReasons = append(result.ApprovalReasons, reason)
			}
		}
	}
	return result
}

func ruleMatchesResource(rule PolicyRule, req ProvisionRequest) bool {
	return rule.ResourceType == "*" || rule.ResourceType == req.ResourceType
}

func resolveAttribute(attr string, req ProvisionRequest) (string, bool) {
	switch attr {
	case "environment":
		return req.Environment, true
	case "resource_type":
		return req.ResourceType, true
	default:
		v, ok := req.Attributes[attr]
		return v, ok
	}
}

func evaluateCondition(op RuleOperator, actual, expected string) bool {
	switch op {
	case OpEquals:
		return actual == expected
	case OpNotEquals:
		return actual != expected
	case OpIn:
		for _, v := range splitCSV(expected) {
			if actual == v {
				return true
			}
		}
		return false
	case OpNotIn:
		for _, v := range splitCSV(expected) {
			if actual == v {
				return false
			}
		}
		return true
	case OpLessThan:
		return actual < expected
	case OpGreaterThan:
		return actual > expected
	case OpLessEq:
		return actual <= expected
	case OpGreaterEq:
		return actual >= expected
	default:
		return false
	}
}

func splitCSV(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
