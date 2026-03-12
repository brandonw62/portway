package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/portway/portway/internal/core"
	"github.com/portway/portway/internal/db"
)

type projectHandler struct {
	q *db.Queries
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// HandleList returns projects for a team (GET /api/v1/teams/{teamId}/projects).
func (h *projectHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	limit, offset := parsePagination(r)

	projects, err := h.q.ListProjectsByTeam(r.Context(), db.ListProjectsByTeamParams{
		TeamID: teamID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, projects)
}

// HandleGet returns a single project (GET /api/v1/teams/{teamId}/projects/{projectId}).
func (h *projectHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	project, err := h.q.GetProject(r.Context(), projectID)
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, project)
}

// HandleCreate creates a new project within a team (POST /api/v1/teams/{teamId}/projects).
func (h *projectHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	slug := slugify(req.Name)
	project, err := h.q.CreateProject(r.Context(), db.CreateProjectParams{
		TeamID:      teamID,
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, fmt.Sprintf("project with slug %q already exists in this team", slug))
			return
		}
		mapError(w, err)
		return
	}

	// Add the creator as project admin.
	user := MustUserFromContext(r.Context())
	h.q.AddMembership(r.Context(), db.AddMembershipParams{
		UserID:    user.ID,
		ProjectID: project.ID,
		Role:      string(core.RoleAdmin),
	})

	respondJSON(w, http.StatusCreated, project)
}

// HandleUpdate updates a project (PUT /api/v1/teams/{teamId}/projects/{projectId}).
func (h *projectHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")

	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	project, err := h.q.UpdateProject(r.Context(), db.UpdateProjectParams{
		ID:          projectID,
		Name:        req.Name,
		Slug:        slugify(req.Name),
		Description: req.Description,
	})
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, project)
}

// HandleDelete deletes a project (DELETE /api/v1/teams/{teamId}/projects/{projectId}).
func (h *projectHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if err := h.q.DeleteProject(r.Context(), projectID); err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleListMembers returns project members (GET /api/v1/teams/{teamId}/projects/{projectId}/members).
func (h *projectHandler) HandleListMembers(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	members, err := h.q.ListProjectMembers(r.Context(), projectID)
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, members)
}

type addProjectMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// HandleAddMember adds a member to a project (POST /api/v1/teams/{teamId}/projects/{projectId}/members).
func (h *projectHandler) HandleAddMember(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")

	var req addProjectMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.UserID == "" || req.Role == "" {
		respondError(w, http.StatusBadRequest, "user_id and role are required")
		return
	}

	_, err := h.q.AddMembership(r.Context(), db.AddMembershipParams{
		UserID:    req.UserID,
		ProjectID: projectID,
		Role:      req.Role,
	})
	if err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleRemoveMember removes a member from a project (DELETE /api/v1/teams/{teamId}/projects/{projectId}/members/{userId}).
func (h *projectHandler) HandleRemoveMember(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	userID := chi.URLParam(r, "userId")

	err := h.q.RemoveMembership(r.Context(), db.RemoveMembershipParams{
		UserID:    userID,
		ProjectID: projectID,
	})
	if err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
