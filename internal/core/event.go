package core

import (
	"encoding/json"
	"time"
)

// EventType classifies provisioning events.
type EventType string

const (
	EventTypeStatusChange EventType = "status_change"
	EventTypeSpecChange   EventType = "spec_change"
	EventTypeError        EventType = "error"
)

// ProvisioningEvent records an audit trail entry for a resource lifecycle change.
type ProvisioningEvent struct {
	ID         string          `json:"id"`
	ResourceID string          `json:"resource_id"`
	Type       EventType       `json:"type"`
	OldStatus  ResourceStatus  `json:"old_status,omitempty"`
	NewStatus  ResourceStatus  `json:"new_status,omitempty"`
	Message    string          `json:"message,omitempty"`
	Detail     json.RawMessage `json:"detail,omitempty"`
	ActorID    string          `json:"actor_id"`
	CreatedAt  time.Time       `json:"created_at"`
}
