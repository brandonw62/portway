package jobs

import "context"

// ProvisionResult is the outcome of a provisioning or deletion operation.
type ProvisionResult struct {
	ProviderRef string // opaque reference in the backing provider (ARN, Terraform ID, etc.)
	Message     string // human-readable status message
}

// Provisioner is the interface that cloud provider integrations must implement.
// The worker delegates actual infrastructure operations to implementations of
// this interface, keeping the job handler logic provider-agnostic.
type Provisioner interface {
	// Provision creates the infrastructure resource described by the given spec.
	// resourceTypeSlug identifies the catalog entry (e.g. "postgres", "redis").
	// spec is the JSON resource configuration.
	Provision(ctx context.Context, resourceTypeSlug string, spec []byte) (*ProvisionResult, error)

	// Delete tears down the infrastructure resource identified by providerRef.
	Delete(ctx context.Context, resourceTypeSlug string, providerRef string) error

	// HealthCheck verifies the resource identified by providerRef is healthy.
	// Returns nil if healthy, an error describing the issue otherwise.
	HealthCheck(ctx context.Context, resourceTypeSlug string, providerRef string) error
}
