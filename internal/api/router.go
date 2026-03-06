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
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// RouterConfig holds the dependencies needed to construct the HTTP router.
type RouterConfig struct {
	Logger zerolog.Logger
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

	// -- Routes -----------------------------------------------------------

	r.Get("/healthz", HandleHealthz)

	// API v1 subrouter — extend here as features are added.
	r.Route("/api/v1", func(r chi.Router) {
		// placeholder: service catalog, component, GitHub webhook routes
	})

	return r
}
