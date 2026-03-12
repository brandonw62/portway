package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

func testLogger() zerolog.Logger {
	return zerolog.Nop()
}

func TestResourceProvisionPayloadMarshal(t *testing.T) {
	p := ResourceProvisionPayload{ResourceID: "r-123", ActorID: "u-456"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ResourceProvisionPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ResourceID != "r-123" || decoded.ActorID != "u-456" {
		t.Fatalf("unexpected decoded payload: %+v", decoded)
	}
}

func TestResourceDeletePayloadMarshal(t *testing.T) {
	p := ResourceDeletePayload{ResourceID: "r-789", ActorID: "u-012"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ResourceDeletePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ResourceID != "r-789" || decoded.ActorID != "u-012" {
		t.Fatalf("unexpected decoded payload: %+v", decoded)
	}
}

func TestResourceHealthCheckPayloadMarshal(t *testing.T) {
	p := ResourceHealthCheckPayload{ResourceID: "r-abc"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ResourceHealthCheckPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ResourceID != "r-abc" {
		t.Fatalf("unexpected decoded payload: %+v", decoded)
	}
}

func TestNoopProvisioner(t *testing.T) {
	p := &NoopProvisioner{}
	ctx := t.Context()

	result, err := p.Provision(ctx, "postgres", []byte(`{"size":"small"}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.ProviderRef == "" {
		t.Fatal("expected non-empty provider ref")
	}

	if err := p.Delete(ctx, "postgres", result.ProviderRef); err != nil {
		t.Fatal(err)
	}

	if err := p.HealthCheck(ctx, "postgres", result.ProviderRef); err != nil {
		t.Fatal(err)
	}
}

func TestNoopProvisionerMultipleResourceTypes(t *testing.T) {
	p := &NoopProvisioner{}
	ctx := t.Context()

	types := []string{"postgres", "redis", "s3", "rabbitmq"}
	for _, rt := range types {
		result, err := p.Provision(ctx, rt, []byte(`{}`))
		if err != nil {
			t.Fatalf("provision %s: %v", rt, err)
		}
		if result.ProviderRef == "" {
			t.Fatalf("expected provider ref for %s", rt)
		}
		if err := p.HealthCheck(ctx, rt, result.ProviderRef); err != nil {
			t.Fatalf("health check %s: %v", rt, err)
		}
		if err := p.Delete(ctx, rt, result.ProviderRef); err != nil {
			t.Fatalf("delete %s: %v", rt, err)
		}
	}
}

func TestDefaultRetryOpts(t *testing.T) {
	opts := DefaultRetryOpts()
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
}

func TestCriticalRetryOpts(t *testing.T) {
	opts := CriticalRetryOpts()
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
}

func TestHandleProvisionInvalidPayload(t *testing.T) {
	h := NewResourceHandler(nil, &NoopProvisioner{}, testLogger())
	task := asynq.NewTask(TypeResourceProvision, []byte(`{invalid json`))
	err := h.HandleProvision(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("expected unmarshal error, got: %v", err)
	}
}

func TestHandleDeleteInvalidPayload(t *testing.T) {
	h := NewResourceHandler(nil, &NoopProvisioner{}, testLogger())
	task := asynq.NewTask(TypeResourceDelete, []byte(`not json`))
	err := h.HandleDelete(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("expected unmarshal error, got: %v", err)
	}
}

func TestHandleHealthCheckInvalidPayload(t *testing.T) {
	h := NewResourceHandler(nil, &NoopProvisioner{}, testLogger())
	task := asynq.NewTask(TypeResourceHealthCheck, []byte(`{{`))
	err := h.HandleHealthCheck(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("expected unmarshal error, got: %v", err)
	}
}

// failingProvisioner is a test provisioner that returns errors.
type failingProvisioner struct {
	provisionErr   error
	deleteErr      error
	healthCheckErr error
}

func (f *failingProvisioner) Provision(_ context.Context, _ string, _ []byte) (*ProvisionResult, error) {
	if f.provisionErr != nil {
		return nil, f.provisionErr
	}
	return &ProvisionResult{ProviderRef: "test://ref", Message: "ok"}, nil
}

func (f *failingProvisioner) Delete(_ context.Context, _ string, _ string) error {
	return f.deleteErr
}

func (f *failingProvisioner) HealthCheck(_ context.Context, _ string, _ string) error {
	return f.healthCheckErr
}

func TestFailingProvisionerInterface(t *testing.T) {
	fp := &failingProvisioner{
		provisionErr: errors.New("cloud API unavailable"),
	}
	var p Provisioner = fp
	_, err := p.Provision(context.Background(), "postgres", []byte(`{}`))
	if err == nil || err.Error() != "cloud API unavailable" {
		t.Fatalf("expected cloud API error, got: %v", err)
	}

	fp.provisionErr = nil
	result, err := p.Provision(context.Background(), "postgres", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.ProviderRef != "test://ref" {
		t.Fatalf("expected test ref, got %s", result.ProviderRef)
	}
}

func TestFailingProvisionerDelete(t *testing.T) {
	fp := &failingProvisioner{deleteErr: errors.New("resource locked")}
	err := fp.Delete(context.Background(), "postgres", "ref-123")
	if err == nil || err.Error() != "resource locked" {
		t.Fatalf("expected resource locked error, got: %v", err)
	}
}

func TestFailingProvisionerHealthCheck(t *testing.T) {
	fp := &failingProvisioner{healthCheckErr: errors.New("connection refused")}
	err := fp.HealthCheck(context.Background(), "redis", "ref-456")
	if err == nil || err.Error() != "connection refused" {
		t.Fatalf("expected connection refused error, got: %v", err)
	}
}

func TestTaskTypeConstants(t *testing.T) {
	types := []string{TypeGitHubSync, TypeResourceProvision, TypeResourceDelete, TypeResourceHealthCheck}
	seen := make(map[string]bool)
	for _, tt := range types {
		if tt == "" {
			t.Fatal("task type constant must not be empty")
		}
		if seen[tt] {
			t.Fatalf("duplicate task type constant: %s", tt)
		}
		seen[tt] = true
	}
}

func TestRegisterHandlers(t *testing.T) {
	mux := asynq.NewServeMux()
	h := NewResourceHandler(nil, &NoopProvisioner{}, testLogger())
	h.Register(mux)
}

func TestPayloadRoundTrips(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{"provision", ResourceProvisionPayload{ResourceID: "r-1", ActorID: "u-1"}},
		{"delete", ResourceDeletePayload{ResourceID: "r-2", ActorID: "u-2"}},
		{"healthcheck", ResourceHealthCheckPayload{ResourceID: "r-3"}},
		{"github_sync", GitHubSyncPayload{InstallationID: 12345}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if len(data) == 0 {
				t.Fatal("expected non-empty JSON")
			}
			var m map[string]any
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatalf("not valid JSON: %v", err)
			}
		})
	}
}
