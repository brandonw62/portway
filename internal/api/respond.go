package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/portway/portway/internal/core"
)

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorResponse struct {
	Error string `json:"error"`
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, errorResponse{Error: msg})
}

// mapError translates a domain or database error into an HTTP status + message.
func mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, core.ErrNotFound):
		respondError(w, http.StatusNotFound, "not found")
	case errors.Is(err, core.ErrConflict):
		respondError(w, http.StatusConflict, "conflict")
	case errors.Is(err, core.ErrValidation):
		respondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, core.ErrUnauthorized):
		respondError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, core.ErrForbidden):
		respondError(w, http.StatusForbidden, "forbidden")
	default:
		respondError(w, http.StatusInternalServerError, "internal error")
	}
}
