package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/portway/portway/internal/core"
	"github.com/portway/portway/internal/db"
	"github.com/portway/portway/internal/jobs"
)

type resourceHandler struct {
	q    *db.Queries
	jobs *jobs.Client
}

type createResourceRequest struct {
	ProjectID      string          `json:"project_id"`
	ResourceTypeID string          `json:"resource_type_id"`
	Name           string          `json:"name"`
	Spec           json.RawMessage `json:"spec"`
}

// HandleCreate provisions a new resource (POST /api/v1/resources).
func (h *resourceHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req createResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.ProjectID == "" || req.ResourceTypeID == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "project_id, resource_type_id, and name are required")
		return
	}

	// Verify the resource type exists and is enabled.
	rt, err := h.q.GetResourceType(r.Context(), req.ResourceTypeID)
	if err != nil {
		mapError(w, err)
		return
	}
	if !rt.Enabled {
		respondError(w, http.StatusBadRequest, "resource type is disabled")
		return
	}

	// Verify the project exists.
	if _, err := h.q.GetProject(r.Context(), req.ProjectID); err != nil {
		mapError(w, err)
		return
	}

	slug := slugify(req.Name)
	spec := req.Spec
	if spec == nil {
		spec = rt.DefaultSpec
	}

	user := MustUserFromContext(r.Context())
	requestedBy := user.ID

	resource, err := h.q.CreateResource(r.Context(), db.CreateResourceParams{
		ProjectID:      req.ProjectID,
		ResourceTypeID: req.ResourceTypeID,
		Name:           req.Name,
		Slug:           slug,
		Status:         string(core.ResourceStatusRequested),
		Spec:           spec,
		RequestedBy:    requestedBy,
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, fmt.Sprintf("resource with slug %q already exists in this project", slug))
			return
		}
		mapError(w, err)
		return
	}

	// Record the initial provisioning event.
	h.q.CreateProvisioningEvent(r.Context(), db.CreateProvisioningEventParams{
		ResourceID: resource.ID,
		Type:       string(core.EventTypeStatusChange),
		OldStatus:  "",
		NewStatus:  string(core.ResourceStatusRequested),
		Message:    "resource provisioning requested",
		Detail:     []byte("{}"),
		ActorID:    requestedBy,
	})

	// Enqueue the async provisioning job.
	if h.jobs != nil {
		h.jobs.EnqueueResourceProvision(r.Context(), jobs.ResourceProvisionPayload{
			ResourceID: resource.ID,
			ActorID:    requestedBy,
		})
	}

	respondJSON(w, http.StatusAccepted, resource)
}

// HandleGet returns a single resource by ID (GET /api/v1/resources/{id}).
func (h *resourceHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	resource, err := h.q.GetResource(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, resource)
}

// HandleList returns resources filtered by project_id and/or status (GET /api/v1/resources).
func (h *resourceHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	// Filter by status takes precedence (used by worker polling).
	if status := r.URL.Query().Get("status"); status != "" {
		resources, err := h.q.ListResourcesByStatus(r.Context(), db.ListResourcesByStatusParams{
			Status: status,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			mapError(w, err)
			return
		}
		respondJSON(w, http.StatusOK, resources)
		return
	}

	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		respondError(w, http.StatusBadRequest, "project_id or status query parameter is required")
		return
	}

	resources, err := h.q.ListResourcesByProject(r.Context(), db.ListResourcesByProjectParams{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, resources)
}

// HandleListByProject returns resources for a specific project (GET /api/v1/projects/{id}/resources).
func (h *resourceHandler) HandleListByProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	limit, offset := parsePagination(r)

	resources, err := h.q.ListResourcesByProject(r.Context(), db.ListResourcesByProjectParams{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, resources)
}

// HandleDelete requests deletion of a resource (DELETE /api/v1/resources/{id}).
func (h *resourceHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resource, err := h.q.GetResource(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	currentStatus := core.ResourceStatus(resource.Status)
	if !currentStatus.CanTransition(core.ResourceStatusDeleting) {
		respondError(w, http.StatusConflict,
			fmt.Sprintf("cannot delete resource in %q status", resource.Status))
		return
	}

	user := MustUserFromContext(r.Context())
	userID := user.ID

	updated, err := h.q.UpdateResourceStatus(r.Context(), db.UpdateResourceStatusParams{
		ID:            id,
		Status:        string(core.ResourceStatusDeleting),
		StatusMessage: "deletion requested",
	})
	if err != nil {
		mapError(w, err)
		return
	}

	h.q.CreateProvisioningEvent(r.Context(), db.CreateProvisioningEventParams{
		ResourceID: id,
		Type:       string(core.EventTypeStatusChange),
		OldStatus:  resource.Status,
		NewStatus:  string(core.ResourceStatusDeleting),
		Message:    "resource deletion requested",
		Detail:     []byte("{}"),
		ActorID:    userID,
	})

	// Enqueue the async deletion job.
	if h.jobs != nil {
		h.jobs.EnqueueResourceDelete(r.Context(), jobs.ResourceDeletePayload{
			ResourceID: id,
			ActorID:    userID,
		})
	}

	respondJSON(w, http.StatusAccepted, updated)
}

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	// Remove anything that isn't alphanumeric or hyphen.
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

func parsePagination(r *http.Request) (limit, offset int32) {
	limit = 50
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	return
}
