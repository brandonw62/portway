// Copyright (C) 2024 Portway Contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// For commercial licensing, contact: licensing@portway.dev

package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// Task type constants. Use these when enqueuing or routing tasks to avoid
// typos and to provide a single source of truth for queue topic names.
const (
	// TypeGitHubSync triggers a full sync of all installed GitHub App repos.
	TypeGitHubSync = "github:sync"

	// TypeResourceProvision provisions a new infrastructure resource.
	TypeResourceProvision = "resource:provision"

	// TypeResourceDelete deprovisions and removes an infrastructure resource.
	TypeResourceDelete = "resource:delete"

	// TypeResourceHealthCheck checks the health of a provisioned resource.
	TypeResourceHealthCheck = "resource:healthcheck"
)

// Client wraps asynq.Client and provides typed enqueue helpers.
type Client struct {
	inner *asynq.Client
}

// NewClient creates an Asynq client connected to the given Redis-compatible URL.
// The returned client must be closed by the caller via Close().
func NewClient(redisURL string) (*Client, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("jobs: failed to parse redis URL: %w", err)
	}
	return &Client{inner: asynq.NewClient(opt)}, nil
}

// Close releases the underlying connection.
func (c *Client) Close() error {
	return c.inner.Close()
}

// GitHubSyncPayload is the JSON payload for a TypeGitHubSync task.
type GitHubSyncPayload struct {
	// InstallationID is the GitHub App installation to sync.
	// A zero value triggers a sync across all known installations.
	InstallationID int64 `json:"installation_id,omitempty"`
}

// EnqueueGitHubSync enqueues a GitHub sync task. opts are passed directly
// to Asynq (e.g. asynq.Queue("critical"), asynq.MaxRetry(3)).
func (c *Client) EnqueueGitHubSync(ctx context.Context, payload GitHubSyncPayload, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("jobs: failed to marshal GitHubSyncPayload: %w", err)
	}
	task := asynq.NewTask(TypeGitHubSync, data, opts...)
	info, err := c.inner.EnqueueContext(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("jobs: failed to enqueue %s: %w", TypeGitHubSync, err)
	}
	return info, nil
}

// ResourceProvisionPayload is the JSON payload for TypeResourceProvision tasks.
type ResourceProvisionPayload struct {
	ResourceID string `json:"resource_id"`
	ActorID    string `json:"actor_id"`
}

// EnqueueResourceProvision enqueues a resource provisioning task.
func (c *Client) EnqueueResourceProvision(ctx context.Context, payload ResourceProvisionPayload, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.enqueue(ctx, TypeResourceProvision, payload, opts...)
}

// ResourceDeletePayload is the JSON payload for TypeResourceDelete tasks.
type ResourceDeletePayload struct {
	ResourceID string `json:"resource_id"`
	ActorID    string `json:"actor_id"`
}

// EnqueueResourceDelete enqueues a resource deletion task.
func (c *Client) EnqueueResourceDelete(ctx context.Context, payload ResourceDeletePayload, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.enqueue(ctx, TypeResourceDelete, payload, opts...)
}

// ResourceHealthCheckPayload is the JSON payload for TypeResourceHealthCheck tasks.
type ResourceHealthCheckPayload struct {
	ResourceID string `json:"resource_id"`
}

// EnqueueResourceHealthCheck enqueues a resource health check task.
func (c *Client) EnqueueResourceHealthCheck(ctx context.Context, payload ResourceHealthCheckPayload, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.enqueue(ctx, TypeResourceHealthCheck, payload, opts...)
}

func (c *Client) enqueue(ctx context.Context, taskType string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("jobs: failed to marshal %s payload: %w", taskType, err)
	}
	task := asynq.NewTask(taskType, data, opts...)
	info, err := c.inner.EnqueueContext(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("jobs: failed to enqueue %s: %w", taskType, err)
	}
	return info, nil
}
