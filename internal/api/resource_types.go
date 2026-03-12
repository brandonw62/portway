package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/portway/portway/internal/db"
)

type resourceTypeHandler struct {
	q *db.Queries
}

// HandleListResourceTypes returns all enabled resource types (the catalog).
func (h *resourceTypeHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	types, err := h.q.ListResourceTypes(r.Context())
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, types)
}

// HandleGetResourceType returns a single resource type by ID.
func (h *resourceTypeHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rt, err := h.q.GetResourceType(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, rt)
}
