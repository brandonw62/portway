package core

import (
	"encoding/json"
	"time"
)

// ResourceStatus represents the lifecycle state of a provisioned resource.
type ResourceStatus string

const (
	ResourceStatusRequested    ResourceStatus = "requested"
	ResourceStatusProvisioning ResourceStatus = "provisioning"
	ResourceStatusReady        ResourceStatus = "ready"
	ResourceStatusUpdating     ResourceStatus = "updating"
	ResourceStatusDeleting     ResourceStatus = "deleting"
	ResourceStatusDeleted      ResourceStatus = "deleted"
	ResourceStatusFailed       ResourceStatus = "failed"
)

// Resource is a provisioned instance of a ResourceType, owned by a project.
type Resource struct {
	ID             string          `json:"id"`
	ProjectID      string          `json:"project_id"`
	ResourceTypeID string          `json:"resource_type_id"`
	Name           string          `json:"name"`
	Slug           string          `json:"slug"`
	Status         ResourceStatus  `json:"status"`
	Spec           json.RawMessage `json:"spec"`
	// ProviderRef is an opaque reference to the resource in the backing provider
	// (e.g., an ARN, a Terraform state ID, a Crossplane external name).
	ProviderRef string `json:"provider_ref,omitempty"`
	// StatusMessage holds a human-readable description of the current status,
	// especially useful for failed states.
	StatusMessage string    `json:"status_message,omitempty"`
	RequestedBy   string    `json:"requested_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ValidTransitions defines which status transitions are allowed.
var ValidTransitions = map[ResourceStatus][]ResourceStatus{
	ResourceStatusRequested:    {ResourceStatusProvisioning, ResourceStatusFailed},
	ResourceStatusProvisioning: {ResourceStatusReady, ResourceStatusFailed},
	ResourceStatusReady:        {ResourceStatusUpdating, ResourceStatusDeleting},
	ResourceStatusUpdating:     {ResourceStatusReady, ResourceStatusFailed},
	ResourceStatusDeleting:     {ResourceStatusDeleted, ResourceStatusFailed},
	ResourceStatusFailed:       {ResourceStatusRequested, ResourceStatusDeleting},
}

// CanTransition reports whether moving from the current status to next is allowed.
func (s ResourceStatus) CanTransition(next ResourceStatus) bool {
	for _, allowed := range ValidTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}
