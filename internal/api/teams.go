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

type teamHandler struct {
	q *db.Queries
}

type createTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// HandleList returns all teams for the authenticated user (GET /api/v1/teams).
func (h *teamHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	user := MustUserFromContext(r.Context())
	teams, err := h.q.ListTeamsForUser(r.Context(), user.ID)
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, teams)
}

// HandleGet returns a single team (GET /api/v1/teams/{teamId}).
func (h *teamHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	team, err := h.q.GetTeam(r.Context(), teamID)
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, team)
}

// HandleCreate creates a new team (POST /api/v1/teams).
func (h *teamHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	slug := slugify(req.Name)
	team, err := h.q.CreateTeam(r.Context(), db.CreateTeamParams{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, fmt.Sprintf("team with slug %q already exists", slug))
			return
		}
		mapError(w, err)
		return
	}

	// Add the creator as team owner.
	user := MustUserFromContext(r.Context())
	h.q.AddTeamMember(r.Context(), db.AddTeamMemberParams{
		TeamID: team.ID,
		UserID: user.ID,
		Role:   string(core.TeamMemberRoleOwner),
	})

	respondJSON(w, http.StatusCreated, team)
}

// HandleUpdate updates a team (PUT /api/v1/teams/{teamId}).
func (h *teamHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	var req updateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	team, err := h.q.UpdateTeam(r.Context(), db.UpdateTeamParams{
		ID:          teamID,
		Name:        req.Name,
		Slug:        slugify(req.Name),
		Description: req.Description,
	})
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, team)
}

// HandleDelete deletes a team (DELETE /api/v1/teams/{teamId}).
func (h *teamHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	if err := h.q.DeleteTeam(r.Context(), teamID); err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleListMembers returns team members (GET /api/v1/teams/{teamId}/members).
func (h *teamHandler) HandleListMembers(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	members, err := h.q.ListTeamMembers(r.Context(), teamID)
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, members)
}

type addTeamMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// HandleAddMember adds a member to a team (POST /api/v1/teams/{teamId}/members).
func (h *teamHandler) HandleAddMember(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	var req addTeamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.UserID == "" || req.Role == "" {
		respondError(w, http.StatusBadRequest, "user_id and role are required")
		return
	}

	err := h.q.AddTeamMember(r.Context(), db.AddTeamMemberParams{
		TeamID: teamID,
		UserID: req.UserID,
		Role:   req.Role,
	})
	if err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleRemoveMember removes a member from a team (DELETE /api/v1/teams/{teamId}/members/{userId}).
func (h *teamHandler) HandleRemoveMember(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	userID := chi.URLParam(r, "userId")

	err := h.q.RemoveTeamMember(r.Context(), db.RemoveTeamMemberParams{
		TeamID: teamID,
		UserID: userID,
	})
	if err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
