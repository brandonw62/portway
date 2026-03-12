package core

import "testing"

func TestCheckQuota_NoQuotaDefined(t *testing.T) {
	usage := CheckQuota(nil, "proj-1", "database", 5)
	if usage != nil {
		t.Fatal("expected nil when no quotas defined")
	}
}

func TestCheckQuota_ProjectScopedTakesPrecedence(t *testing.T) {
	quotas := []Quota{
		{ID: "global", ResourceType: "database", Limit: 10},
		{ID: "proj", ProjectID: "proj-1", ResourceType: "database", Limit: 3},
	}
	usage := CheckQuota(quotas, "proj-1", "database", 2)
	if usage == nil {
		t.Fatal("expected usage result")
	}
	if usage.QuotaID != "proj" {
		t.Fatalf("expected project quota, got %s", usage.QuotaID)
	}
	if usage.Remaining() != 1 {
		t.Fatalf("expected 1 remaining, got %d", usage.Remaining())
	}
	if usage.Exceeded() {
		t.Fatal("should not be exceeded with 2/3")
	}
}

func TestCheckQuota_Exceeded(t *testing.T) {
	quotas := []Quota{
		{ID: "q1", ProjectID: "proj-1", ResourceType: "database", Limit: 3},
	}
	usage := CheckQuota(quotas, "proj-1", "database", 3)
	if !usage.Exceeded() {
		t.Fatal("expected exceeded at 3/3")
	}
	if usage.Remaining() != 0 {
		t.Fatalf("expected 0 remaining, got %d", usage.Remaining())
	}
}

func TestCheckQuota_WildcardResource(t *testing.T) {
	quotas := []Quota{
		{ID: "q1", ProjectID: "proj-1", ResourceType: "*", Limit: 20},
	}
	usage := CheckQuota(quotas, "proj-1", "anything", 15)
	if usage == nil {
		t.Fatal("expected wildcard quota to match")
	}
	if usage.Remaining() != 5 {
		t.Fatalf("expected 5 remaining, got %d", usage.Remaining())
	}
}
