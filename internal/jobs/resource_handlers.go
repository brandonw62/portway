package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/portway/portway/internal/core"
	"github.com/portway/portway/internal/db"
)

// ResourceHandler processes async resource provisioning tasks.
type ResourceHandler struct {
	queries     *db.Queries
	provisioner Provisioner
	logger      zerolog.Logger
}

// NewResourceHandler creates a handler wired to the given DB queries and provisioner.
func NewResourceHandler(queries *db.Queries, provisioner Provisioner, logger zerolog.Logger) *ResourceHandler {
	return &ResourceHandler{
		queries:     queries,
		provisioner: provisioner,
		logger:      logger,
	}
}

// DefaultRetryOpts returns Asynq options with exponential backoff for resource tasks.
func DefaultRetryOpts() []asynq.Option {
	return []asynq.Option{
		asynq.MaxRetry(5),
		asynq.Queue("default"),
	}
}

// CriticalRetryOpts returns Asynq options for critical resource tasks (provisioning).
func CriticalRetryOpts() []asynq.Option {
	return []asynq.Option{
		asynq.MaxRetry(8),
		asynq.Queue("critical"),
	}
}

// Register adds all resource task handlers to the given mux.
func (h *ResourceHandler) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeResourceProvision, h.HandleProvision)
	mux.HandleFunc(TypeResourceDelete, h.HandleDelete)
	mux.HandleFunc(TypeResourceHealthCheck, h.HandleHealthCheck)
}

// HandleProvision processes a TypeResourceProvision task.
func (h *ResourceHandler) HandleProvision(ctx context.Context, t *asynq.Task) error {
	var payload ResourceProvisionPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal provision payload: %w", err)
	}

	log := h.logger.With().
		Str("task", TypeResourceProvision).
		Str("resource_id", payload.ResourceID).
		Logger()

	// Fetch the resource.
	resource, err := h.queries.GetResource(ctx, payload.ResourceID)
	if err != nil {
		return fmt.Errorf("get resource %s: %w", payload.ResourceID, err)
	}

	// Guard: only provision resources in "requested" status.
	if !core.ResourceStatus(resource.Status).CanTransition(core.ResourceStatusProvisioning) {
		log.Warn().Str("status", resource.Status).Msg("resource not in a provisionable state, skipping")
		return nil
	}

	// Resolve the resource type.
	rt, err := h.queries.GetResourceType(ctx, resource.ResourceTypeID)
	if err != nil {
		h.failResource(ctx, resource.ID, payload.ActorID, resource.Status, fmt.Sprintf("failed to look up resource type: %v", err))
		return fmt.Errorf("get resource type %s: %w", resource.ResourceTypeID, err)
	}

	// -- Policy evaluation ------------------------------------------------
	policyResult, err := h.evaluatePolicies(ctx, resource, rt)
	if err != nil {
		log.Error().Err(err).Msg("policy evaluation failed, proceeding without policy check")
		// Non-fatal: if we can't load policies, allow provisioning to continue
		// rather than blocking all work. Audit the failure.
	} else if policyResult != nil {
		h.auditPolicyResult(ctx, payload.ActorID, resource, policyResult)

		if policyResult.Effect == core.PolicyDeny {
			msg := fmt.Sprintf("denied by policy: %v", policyResult.DenyReasons)
			h.failResource(ctx, resource.ID, payload.ActorID, resource.Status, msg)
			log.Warn().Strs("reasons", policyResult.DenyReasons).Msg("provisioning denied by policy")
			return nil // permanent denial — don't retry
		}

		if policyResult.Effect == core.PolicyRequireApproval {
			if err := h.createApprovalRequest(ctx, resource, rt, payload.ActorID, policyResult); err != nil {
				log.Error().Err(err).Msg("failed to create approval request")
				return fmt.Errorf("create approval request: %w", err)
			}
			log.Info().Strs("reasons", policyResult.ApprovalReasons).Msg("provisioning requires approval, request created")
			return nil // will be re-enqueued when approval is granted
		}
	}

	// -- Quota check ------------------------------------------------------
	if exceeded, reason := h.checkQuota(ctx, resource, rt); exceeded {
		msg := fmt.Sprintf("quota exceeded: %s", reason)
		h.failResource(ctx, resource.ID, payload.ActorID, resource.Status, msg)
		h.audit(ctx, payload.ActorID, &resource.ProjectID, string(core.AuditQuotaExceeded), "resource", resource.ID, map[string]any{"reason": reason}, false)
		log.Warn().Str("reason", reason).Msg("provisioning blocked by quota")
		return nil // permanent — don't retry
	}

	// -- Transition to provisioning ---------------------------------------
	resource, err = h.queries.UpdateResourceStatus(ctx, db.UpdateResourceStatusParams{
		ID:            resource.ID,
		Status:        string(core.ResourceStatusProvisioning),
		StatusMessage: "provisioning started",
	})
	if err != nil {
		return fmt.Errorf("update status to provisioning: %w", err)
	}
	h.recordEvent(ctx, resource.ID, payload.ActorID, string(core.ResourceStatusRequested), string(core.ResourceStatusProvisioning), "provisioning started")

	// -- Call the provisioner ---------------------------------------------
	result, err := h.provisioner.Provision(ctx, rt.Slug, resource.Spec)
	if err != nil {
		h.failResource(ctx, resource.ID, payload.ActorID, string(core.ResourceStatusProvisioning), fmt.Sprintf("provision failed: %v", err))
		return fmt.Errorf("provision resource %s: %w", payload.ResourceID, err)
	}

	// Store provider reference.
	if result.ProviderRef != "" {
		if err := h.queries.SetResourceProviderRef(ctx, db.SetResourceProviderRefParams{
			ID:          resource.ID,
			ProviderRef: result.ProviderRef,
		}); err != nil {
			return fmt.Errorf("set provider ref: %w", err)
		}
	}

	// Transition to ready.
	msg := "resource ready"
	if result.Message != "" {
		msg = result.Message
	}
	_, err = h.queries.UpdateResourceStatus(ctx, db.UpdateResourceStatusParams{
		ID:            resource.ID,
		Status:        string(core.ResourceStatusReady),
		StatusMessage: msg,
	})
	if err != nil {
		return fmt.Errorf("update status to ready: %w", err)
	}
	h.recordEvent(ctx, resource.ID, payload.ActorID, string(core.ResourceStatusProvisioning), string(core.ResourceStatusReady), msg)
	h.audit(ctx, payload.ActorID, &resource.ProjectID, string(core.AuditResourceProvision), "resource", resource.ID, map[string]any{"provider_ref": result.ProviderRef}, true)

	log.Info().Str("provider_ref", result.ProviderRef).Msg("resource provisioned successfully")
	return nil
}

// evaluatePolicies loads active policies for the resource's project and evaluates them.
func (h *ResourceHandler) evaluatePolicies(ctx context.Context, resource db.Resource, rt db.ResourceType) (*core.EvalResult, error) {
	dbPolicies, err := h.queries.ListActivePoliciesForProject(ctx, &resource.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	if len(dbPolicies) == 0 {
		return nil, nil
	}

	// Load rules for each policy and convert to core types.
	var policies []core.Policy
	for _, dp := range dbPolicies {
		dbRules, err := h.queries.ListPolicyRules(ctx, dp.ID)
		if err != nil {
			return nil, fmt.Errorf("list rules for policy %s: %w", dp.ID, err)
		}
		var rules []core.PolicyRule
		for _, dr := range dbRules {
			rules = append(rules, core.PolicyRule{
				ID:           dr.ID,
				Description:  dr.Description,
				ResourceType: dr.ResourceType,
				Attribute:    dr.Attribute,
				Operator:     core.RuleOperator(dr.Operator),
				Value:        dr.Value,
				Effect:       core.PolicyEffect(dr.Effect),
			})
		}
		var projectID string
		if dp.ProjectID != nil {
			projectID = *dp.ProjectID
		}
		policies = append(policies, core.Policy{
			ID:        dp.ID,
			Name:      dp.Name,
			Scope:     core.PolicyScope(dp.Scope),
			ProjectID: projectID,
			Rules:     rules,
			Enabled:   dp.Enabled,
		})
	}

	// Build the provision request from resource attributes.
	attrs := make(map[string]string)
	// Parse resource spec into flat attributes for policy evaluation.
	var specMap map[string]any
	if err := json.Unmarshal(resource.Spec, &specMap); err == nil {
		for k, v := range specMap {
			attrs[k] = fmt.Sprintf("%v", v)
		}
	}
	attrs["resource_type_slug"] = rt.Slug
	attrs["resource_type_category"] = rt.Category

	provReq := core.ProvisionRequest{
		UserID:       resource.RequestedBy,
		ProjectID:    resource.ProjectID,
		ResourceType: rt.Category,
		Environment:  attrs["environment"],
		Attributes:   attrs,
	}

	result := core.EvaluatePolicies(policies, provReq)
	return &result, nil
}

// checkQuota verifies that the project hasn't exceeded its resource quota.
// Returns true if quota is exceeded, along with a reason string.
func (h *ResourceHandler) checkQuota(ctx context.Context, resource db.Resource, rt db.ResourceType) (bool, string) {
	dbQuotas, err := h.queries.ListQuotasForProject(ctx, &resource.ProjectID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to load quotas, skipping quota check")
		return false, ""
	}
	if len(dbQuotas) == 0 {
		return false, ""
	}

	// Convert DB quotas to core types.
	var quotas []core.Quota
	for _, dq := range dbQuotas {
		var projectID string
		if dq.ProjectID != nil {
			projectID = *dq.ProjectID
		}
		quotas = append(quotas, core.Quota{
			ID:           dq.ID,
			ProjectID:    projectID,
			ResourceType: dq.ResourceType,
			Limit:        int(dq.Limit),
		})
	}

	// Get current count of active resources of this type in the project.
	count, err := h.queries.CountResourcesByProjectAndType(ctx, db.CountResourcesByProjectAndTypeParams{
		ProjectID:      resource.ProjectID,
		ResourceTypeID: resource.ResourceTypeID,
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to count resources, skipping quota check")
		return false, ""
	}

	usage := core.CheckQuota(quotas, resource.ProjectID, rt.Category, int(count))
	if usage != nil && usage.Exceeded() {
		return true, fmt.Sprintf("%s limit %d reached (%d in use)", rt.Category, usage.Limit, usage.Current)
	}
	return false, ""
}

// createApprovalRequest persists an approval request when policy requires it.
func (h *ResourceHandler) createApprovalRequest(ctx context.Context, resource db.Resource, rt db.ResourceType, actorID string, result *core.EvalResult) error {
	payload, _ := json.Marshal(map[string]any{
		"resource_id":   resource.ID,
		"resource_type": rt.Slug,
		"project_id":    resource.ProjectID,
		"spec":          json.RawMessage(resource.Spec),
	})
	reasons, _ := json.Marshal(result.ApprovalReasons)
	matchedPolicies, _ := json.Marshal(result.MatchedPolicies)

	expiresAt := pgtype.Timestamptz{Time: time.Now().Add(72 * time.Hour), Valid: true}

	_, err := h.queries.CreateApprovalRequest(ctx, db.CreateApprovalRequestParams{
		ProjectID:       resource.ProjectID,
		RequestedBy:     actorID,
		ResourceType:    rt.Category,
		RequestPayload:  payload,
		Reasons:         reasons,
		MatchedPolicies: matchedPolicies,
		Status:          string(core.ApprovalPending),
		ExpiresAt:       expiresAt,
	})
	if err != nil {
		return err
	}

	h.audit(ctx, actorID, &resource.ProjectID, string(core.AuditApprovalRequested), "resource", resource.ID,
		map[string]any{"reasons": result.ApprovalReasons, "matched_policies": result.MatchedPolicies}, false)
	return nil
}

// auditPolicyResult records the outcome of policy evaluation.
func (h *ResourceHandler) auditPolicyResult(ctx context.Context, actorID string, resource db.Resource, result *core.EvalResult) {
	action := string(core.AuditPolicyEvaluated)
	if result.Effect == core.PolicyDeny {
		action = string(core.AuditPolicyDenied)
	}
	h.audit(ctx, actorID, &resource.ProjectID, action, "resource", resource.ID,
		map[string]any{
			"effect":           string(result.Effect),
			"matched_policies": result.MatchedPolicies,
			"deny_reasons":     result.DenyReasons,
			"approval_reasons": result.ApprovalReasons,
		}, result.Allowed)
}

// HandleDelete processes a TypeResourceDelete task.
func (h *ResourceHandler) HandleDelete(ctx context.Context, t *asynq.Task) error {
	var payload ResourceDeletePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal delete payload: %w", err)
	}

	log := h.logger.With().
		Str("task", TypeResourceDelete).
		Str("resource_id", payload.ResourceID).
		Logger()

	resource, err := h.queries.GetResource(ctx, payload.ResourceID)
	if err != nil {
		return fmt.Errorf("get resource %s: %w", payload.ResourceID, err)
	}

	if !core.ResourceStatus(resource.Status).CanTransition(core.ResourceStatusDeleting) {
		log.Warn().Str("status", resource.Status).Msg("resource not in a deletable state, skipping")
		return nil
	}

	// Transition to deleting.
	oldStatus := resource.Status
	resource, err = h.queries.UpdateResourceStatus(ctx, db.UpdateResourceStatusParams{
		ID:            resource.ID,
		Status:        string(core.ResourceStatusDeleting),
		StatusMessage: "deletion started",
	})
	if err != nil {
		return fmt.Errorf("update status to deleting: %w", err)
	}
	h.recordEvent(ctx, resource.ID, payload.ActorID, oldStatus, string(core.ResourceStatusDeleting), "deletion started")

	// Resolve resource type for the provisioner.
	rt, err := h.queries.GetResourceType(ctx, resource.ResourceTypeID)
	if err != nil {
		h.failResource(ctx, resource.ID, payload.ActorID, string(core.ResourceStatusDeleting), fmt.Sprintf("failed to look up resource type: %v", err))
		return fmt.Errorf("get resource type %s: %w", resource.ResourceTypeID, err)
	}

	// Call the provisioner.
	if err := h.provisioner.Delete(ctx, rt.Slug, resource.ProviderRef); err != nil {
		h.failResource(ctx, resource.ID, payload.ActorID, string(core.ResourceStatusDeleting), fmt.Sprintf("delete failed: %v", err))
		return fmt.Errorf("delete resource %s: %w", payload.ResourceID, err)
	}

	// Transition to deleted.
	_, err = h.queries.UpdateResourceStatus(ctx, db.UpdateResourceStatusParams{
		ID:            resource.ID,
		Status:        string(core.ResourceStatusDeleted),
		StatusMessage: "resource deleted",
	})
	if err != nil {
		return fmt.Errorf("update status to deleted: %w", err)
	}
	h.recordEvent(ctx, resource.ID, payload.ActorID, string(core.ResourceStatusDeleting), string(core.ResourceStatusDeleted), "resource deleted")

	h.audit(ctx, payload.ActorID, &resource.ProjectID, string(core.AuditResourceDeprovision), "resource", resource.ID, map[string]any{"provider_ref": resource.ProviderRef}, true)

	log.Info().Msg("resource deleted successfully")
	return nil
}

// HandleHealthCheck processes a TypeResourceHealthCheck task.
func (h *ResourceHandler) HandleHealthCheck(ctx context.Context, t *asynq.Task) error {
	var payload ResourceHealthCheckPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal healthcheck payload: %w", err)
	}

	resource, err := h.queries.GetResource(ctx, payload.ResourceID)
	if err != nil {
		return fmt.Errorf("get resource %s: %w", payload.ResourceID, err)
	}

	// Only health-check resources that are ready.
	if resource.Status != string(core.ResourceStatusReady) {
		return nil
	}

	rt, err := h.queries.GetResourceType(ctx, resource.ResourceTypeID)
	if err != nil {
		return fmt.Errorf("get resource type %s: %w", resource.ResourceTypeID, err)
	}

	if err := h.provisioner.HealthCheck(ctx, rt.Slug, resource.ProviderRef); err != nil {
		h.logger.Warn().
			Str("resource_id", payload.ResourceID).
			Err(err).
			Msg("health check failed")
		_, _ = h.queries.UpdateResourceStatus(ctx, db.UpdateResourceStatusParams{
			ID:            resource.ID,
			Status:        string(core.ResourceStatusReady),
			StatusMessage: fmt.Sprintf("health check failed: %v", err),
		})
		return fmt.Errorf("healthcheck resource %s: %w", payload.ResourceID, err)
	}

	return nil
}

// failResource transitions a resource to failed status and records an event.
func (h *ResourceHandler) failResource(ctx context.Context, resourceID, actorID, oldStatus, message string) {
	_, err := h.queries.UpdateResourceStatus(ctx, db.UpdateResourceStatusParams{
		ID:            resourceID,
		Status:        string(core.ResourceStatusFailed),
		StatusMessage: message,
	})
	if err != nil {
		h.logger.Error().Err(err).Str("resource_id", resourceID).Msg("failed to update resource to failed status")
	}
	h.recordEvent(ctx, resourceID, actorID, oldStatus, string(core.ResourceStatusFailed), message)
}

// audit writes a platform-level audit entry.
func (h *ResourceHandler) audit(ctx context.Context, actorID string, projectID *string, action, targetType, targetID string, detail map[string]any, allowed bool) {
	detailJSON, _ := json.Marshal(detail)
	_, err := h.queries.CreateAuditEntry(ctx, db.CreateAuditEntryParams{
		ActorID:    actorID,
		ProjectID:  projectID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detailJSON,
		Allowed:    allowed,
	})
	if err != nil {
		h.logger.Error().Err(err).Str("action", action).Str("target_id", targetID).Msg("failed to record audit entry")
	}
}

// recordEvent writes a provisioning event to the audit trail.
func (h *ResourceHandler) recordEvent(ctx context.Context, resourceID, actorID, oldStatus, newStatus, message string) {
	_, err := h.queries.CreateProvisioningEvent(ctx, db.CreateProvisioningEventParams{
		ResourceID: resourceID,
		Type:       string(core.EventTypeStatusChange),
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
		Message:    message,
		Detail:     []byte("{}"),
		ActorID:    actorID,
	})
	if err != nil {
		h.logger.Error().Err(err).
			Str("resource_id", resourceID).
			Str("old_status", oldStatus).
			Str("new_status", newStatus).
			Msg("failed to record provisioning event")
	}
}
