package core

import "time"

// Quota defines a resource limit scoped to a project (or globally when ProjectID is empty).
type Quota struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id,omitempty"` // empty = global default
	ResourceType string    `json:"resource_type"`        // e.g. "database", "compute", "*"
	Limit        int       `json:"limit"`                // max count of this resource type
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// QuotaUsage tracks current consumption against a quota.
type QuotaUsage struct {
	QuotaID   string `json:"quota_id"`
	ProjectID string `json:"project_id"`
	Current   int    `json:"current"`
	Limit     int    `json:"limit"`
}

// Exceeded returns true if current usage meets or exceeds the limit.
func (u QuotaUsage) Exceeded() bool {
	return u.Current >= u.Limit
}

// Remaining returns how many more resources can be provisioned.
func (u QuotaUsage) Remaining() int {
	r := u.Limit - u.Current
	if r < 0 {
		return 0
	}
	return r
}

// CheckQuota evaluates whether provisioning one more resource of the given
// type would exceed the applicable quota. Returns nil if within limits.
func CheckQuota(quotas []Quota, projectID, resourceType string, currentCount int) *QuotaUsage {
	// Find the most specific quota: project-scoped first, then global.
	var matched *Quota
	for i := range quotas {
		q := &quotas[i]
		if q.ResourceType != resourceType && q.ResourceType != "*" {
			continue
		}
		if q.ProjectID == projectID {
			// Project-specific quota takes precedence.
			matched = q
			break
		}
		if q.ProjectID == "" && matched == nil {
			matched = q
		}
	}
	if matched == nil {
		return nil // no quota defined — unlimited
	}
	usage := &QuotaUsage{
		QuotaID:   matched.ID,
		ProjectID: projectID,
		Current:   currentCount,
		Limit:     matched.Limit,
	}
	return usage
}
