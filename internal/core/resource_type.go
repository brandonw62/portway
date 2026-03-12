package core

import (
	"encoding/json"
	"time"
)

// ResourceTypeCategory groups resource types into broad categories for UI and policy purposes.
type ResourceTypeCategory string

const (
	ResourceTypeCategoryDatabase  ResourceTypeCategory = "database"
	ResourceTypeCategoryCache     ResourceTypeCategory = "cache"
	ResourceTypeCategoryStorage   ResourceTypeCategory = "storage"
	ResourceTypeCategoryMessaging ResourceTypeCategory = "messaging"
	ResourceTypeCategoryNetwork   ResourceTypeCategory = "network"
	ResourceTypeCategorySecret    ResourceTypeCategory = "secret"
)

// ResourceType defines a kind of infrastructure resource that developers can provision
// (e.g., PostgreSQL database, Redis cache, S3 bucket). This is the catalog entry.
type ResourceType struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Slug        string               `json:"slug"`
	Category    ResourceTypeCategory `json:"category"`
	Description string               `json:"description,omitempty"`
	// DefaultSpec is the default configuration applied when a developer provisions
	// a resource of this type without specifying all fields.
	DefaultSpec json.RawMessage `json:"default_spec,omitempty"`
	// SpecSchema is a JSON Schema that validates the spec provided at provisioning time.
	SpecSchema json.RawMessage `json:"spec_schema,omitempty"`
	Enabled    bool            `json:"enabled"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}
