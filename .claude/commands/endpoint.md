Add a new API endpoint to the Portway server.

1. Parse $ARGUMENTS for the HTTP method, path, and purpose (e.g., "GET /api/v1/teams list all teams")
2. Create or update the handler file in `internal/api/` (group by resource: `teams.go`, `projects.go`, etc.)
3. Define a handler struct if one doesn't exist for this resource, with `q *db.Queries` and optionally `jobs *jobs.Client`
4. Implement the handler method following existing patterns in `internal/api/resources.go`:
   - Extract user identity via `UserFromContext(r.Context())`
   - Use `respondJSON()` for success responses
   - Use `respondError()` for errors, mapping `core.ErrNotFound` → 404, `core.ErrForbidden` → 403, etc.
5. Register the route in `internal/api/router.go` under the `/api/v1` group (inside the auth middleware block)
6. Run `go build ./cmd/portway` to verify it compiles
