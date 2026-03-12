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

package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"

	"github.com/portway/portway/internal/db"
	"github.com/portway/portway/internal/jobs"
)

// RouterConfig holds the dependencies needed to construct the HTTP router.
type RouterConfig struct {
	Logger      zerolog.Logger
	Queries     *db.Queries
	Jobs        *jobs.Client
	Auth        AuthConfig
}

// NewRouter constructs and returns the application's root Chi router.
// Middleware is applied in the following order:
//  1. hlog request ID injection (propagates via context)
//  2. hlog access logger (structured JSON via zerolog)
//  3. Chi Recoverer (catches panics, returns 500)
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// -- Middleware stack -------------------------------------------------

	// Inject a unique request ID into every request context and response header.
	r.Use(hlog.NewHandler(cfg.Logger))
	r.Use(hlog.RequestIDHandler("request_id", "X-Request-Id"))

	// Structured access log line per request.
	r.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("request")
	}))

	// Recover from panics and return HTTP 500.
	r.Use(middleware.Recoverer)

	// -- Auth setup -------------------------------------------------------

	var authMW func(http.Handler) http.Handler
	var ah *authHandler

	if cfg.Auth.OIDCEnabled() {
		var err error
		ah, err = newAuthHandler(context.Background(), cfg.Auth)
		if err != nil {
			cfg.Logger.Fatal().Err(err).Msg("failed to initialize OIDC provider")
		}
		authMW = ah.AuthMiddleware
		cfg.Logger.Info().Str("issuer", cfg.Auth.IssuerURL).Msg("OIDC authentication enabled")
	} else if cfg.Auth.Environment == "development" {
		authMW = devAuthMiddleware(cfg.Queries)
		cfg.Logger.Warn().Msg("OIDC not configured — using X-User-Id header auth (development mode)")
	}

	// -- Routes -----------------------------------------------------------

	r.Get("/healthz", HandleHealthz)

	// API v1 subrouter.
	r.Route("/api/v1", func(r chi.Router) {
		// Auth endpoints (unauthenticated).
		if ah != nil {
			r.Get("/auth/login", ah.HandleLogin)
			r.Get("/auth/callback", ah.HandleCallback)
			r.Get("/auth/me", ah.HandleMe)
		} else if cfg.Auth.Environment == "development" {
			r.Get("/auth/me", devHandleMe(cfg.Queries))
		}

		// Authenticated routes in a separate group so middleware is
		// applied before any route registration (Chi requirement).
		r.Group(func(r chi.Router) {
			if authMW != nil {
				r.Use(authMW)
			}

			rth := &resourceTypeHandler{q: cfg.Queries}
			r.Get("/resource-types", rth.HandleList)
			r.Get("/resource-types/{id}", rth.HandleGet)

			rh := &resourceHandler{q: cfg.Queries, jobs: cfg.Jobs}
			r.Post("/resources", rh.HandleCreate)
			r.Get("/resources", rh.HandleList)
			r.Get("/resources/{id}", rh.HandleGet)
			r.Delete("/resources/{id}", rh.HandleDelete)

			r.Get("/projects/{id}/resources", rh.HandleListByProject)

			apph := &approvalHandler{q: cfg.Queries, jobs: cfg.Jobs}
			r.Get("/approvals", apph.HandleList)
			r.Get("/approvals/{id}", apph.HandleGet)
			r.Post("/approvals/{id}/review", apph.HandleReview)
		})
	})

	return r
}
