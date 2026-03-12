package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/hlog"

	"github.com/portway/portway/internal/core"
	"github.com/portway/portway/internal/db"
	"github.com/portway/portway/internal/jobs"
)

type approvalHandler struct {
	q    *db.Queries
	jobs *jobs.Client
}

// HandleList returns pending approval requests (GET /api/v1/approvals).
func (h *approvalHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	user := MustUserFromContext(r.Context())

	if !h.userCanReview(r, user.ID) {
		respondError(w, http.StatusForbidden, "approval:review permission required")
		return
	}

	limit, offset := parsePagination(r)
	approvals, err := h.q.ListPendingApprovals(r.Context(), db.ListPendingApprovalsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, approvals)
}

// HandleGet returns a single approval request (GET /api/v1/approvals/{id}).
func (h *approvalHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	user := MustUserFromContext(r.Context())

	if !h.userCanReview(r, user.ID) {
		respondError(w, http.StatusForbidden, "approval:review permission required")
		return
	}

	id := chi.URLParam(r, "id")
	approval, err := h.q.GetApprovalRequest(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, approval)
}

type reviewRequest struct {
	Decision string `json:"decision"` // "approved" or "denied"
	Comment  string `json:"comment"`
}

// HandleReview approves or denies an approval request (POST /api/v1/approvals/{id}/review).
func (h *approvalHandler) HandleReview(w http.ResponseWriter, r *http.Request) {
	user := MustUserFromContext(r.Context())

	if !h.userCanReview(r, user.ID) {
		respondError(w, http.StatusForbidden, "approval:review permission required")
		return
	}

	id := chi.URLParam(r, "id")

	var req reviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Decision != string(core.ApprovalApproved) && req.Decision != string(core.ApprovalDenied) {
		respondError(w, http.StatusBadRequest, "decision must be \"approved\" or \"denied\"")
		return
	}

	// Fetch the approval to ensure it's still pending.
	approval, err := h.q.GetApprovalRequest(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}
	if approval.Status != string(core.ApprovalPending) {
		respondError(w, http.StatusConflict, "approval request is no longer pending")
		return
	}

	reviewedBy := user.ID
	updated, err := h.q.ReviewApprovalRequest(r.Context(), db.ReviewApprovalRequestParams{
		ID:            id,
		Status:        req.Decision,
		ReviewedBy:    &reviewedBy,
		ReviewComment: req.Comment,
	})
	if err != nil {
		mapError(w, err)
		return
	}

	log := hlog.FromRequest(r)

	// Parse the request_payload to extract resource_id for downstream actions.
	var payload struct {
		ResourceID string `json:"resource_id"`
	}
	_ = json.Unmarshal(approval.RequestPayload, &payload)

	if req.Decision == string(core.ApprovalApproved) && payload.ResourceID != "" {
		// Re-enqueue the resource provision job so the worker picks it up.
		// First reset the resource status back to "requested" so the worker will process it.
		_, err := h.q.UpdateResourceStatus(r.Context(), db.UpdateResourceStatusParams{
			ID:            payload.ResourceID,
			Status:        string(core.ResourceStatusRequested),
			StatusMessage: "approved — re-queued for provisioning",
		})
		if err != nil {
			log.Error().Err(err).Str("resource_id", payload.ResourceID).Msg("failed to reset resource status after approval")
		}

		if h.jobs != nil {
			h.jobs.EnqueueResourceProvision(r.Context(), jobs.ResourceProvisionPayload{
				ResourceID: payload.ResourceID,
				ActorID:    user.ID,
			})
		}
		log.Info().Str("approval_id", id).Str("resource_id", payload.ResourceID).Msg("approval granted, resource re-queued for provisioning")
	}

	if req.Decision == string(core.ApprovalDenied) && payload.ResourceID != "" {
		// Update resource status to failed with denial reason.
		msg := "denied by reviewer"
		if req.Comment != "" {
			msg = "denied: " + req.Comment
		}
		_, err := h.q.UpdateResourceStatus(r.Context(), db.UpdateResourceStatusParams{
			ID:            payload.ResourceID,
			Status:        string(core.ResourceStatusFailed),
			StatusMessage: msg,
		})
		if err != nil {
			log.Error().Err(err).Str("resource_id", payload.ResourceID).Msg("failed to update resource status after denial")
		}
		log.Info().Str("approval_id", id).Str("resource_id", payload.ResourceID).Msg("approval denied, resource marked as failed")
	}

	// Notification placeholder — log the review action.
	log.Info().
		Str("approval_id", id).
		Str("decision", req.Decision).
		Str("reviewer", user.ID).
		Msg("approval reviewed — notification hooks would fire here (Slack/email)")

	respondJSON(w, http.StatusOK, updated)
}

// userCanReview checks if the user has approval:review permission on any project.
// For now, we check membership on the approval's project. For list endpoints,
// we check if the user is admin or project-owner on any project.
func (h *approvalHandler) userCanReview(r *http.Request, userID string) bool {
	// Check if user has any membership with approval:review permission.
	// Admin and project-owner roles have this permission.
	teams, err := h.q.ListTeamsForUser(r.Context(), userID)
	if err != nil {
		return false
	}
	// If user belongs to any team, check their memberships.
	// For simplicity: list all projects for the user's teams and check membership roles.
	for _, team := range teams {
		projects, err := h.q.ListProjectsByTeam(r.Context(), db.ListProjectsByTeamParams{
			TeamID: team.ID,
			Limit:  100,
			Offset: 0,
		})
		if err != nil {
			continue
		}
		for _, project := range projects {
			membership, err := h.q.GetMembership(r.Context(), db.GetMembershipParams{
				UserID:    userID,
				ProjectID: project.ID,
			})
			if err != nil {
				continue
			}
			if core.HasPermission(core.Role(membership.Role), core.PermApprovalReview) {
				return true
			}
		}
	}
	return false
}
