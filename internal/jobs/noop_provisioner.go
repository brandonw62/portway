package jobs

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

// NoopProvisioner is a placeholder that logs operations without touching real
// infrastructure. Use it during development and testing.
type NoopProvisioner struct{}

func (n *NoopProvisioner) Provision(ctx context.Context, resourceTypeSlug string, spec []byte) (*ProvisionResult, error) {
	log.Ctx(ctx).Info().
		Str("resource_type", resourceTypeSlug).
		RawJSON("spec", spec).
		Msg("noop: simulating provision")
	return &ProvisionResult{
		ProviderRef: fmt.Sprintf("noop://%s/%s", resourceTypeSlug, "simulated"),
		Message:     "provisioned (noop)",
	}, nil
}

func (n *NoopProvisioner) Delete(ctx context.Context, resourceTypeSlug string, providerRef string) error {
	log.Ctx(ctx).Info().
		Str("resource_type", resourceTypeSlug).
		Str("provider_ref", providerRef).
		Msg("noop: simulating delete")
	return nil
}

func (n *NoopProvisioner) HealthCheck(ctx context.Context, resourceTypeSlug string, providerRef string) error {
	log.Ctx(ctx).Info().
		Str("resource_type", resourceTypeSlug).
		Str("provider_ref", providerRef).
		Msg("noop: simulating health check")
	return nil
}
