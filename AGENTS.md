# AGENTS.md

Specialized agent definitions for Claude Code when working on Portway.

## backend

Go backend specialist. Works in `cmd/`, `internal/api/`, `internal/core/`, `internal/jobs/`, `internal/integrations/`.

**Knows:**
- Chi router patterns, middleware chain in `internal/api/router.go`
- Sentinel error convention: return `core.ErrNotFound` etc, map to HTTP in handlers via `respondError()` in `internal/api/respond.go`
- Auth: OIDC via `coreos/go-oidc/v3` with dev-mode fallback (`X-User-Id` header). Identity extracted via `UserFromContext()` in `internal/api/context.go`
- Roles: admin, project-owner, developer, viewer
- Resource lifecycle: requested → provisioning → ready → updating → deleting → deleted (+ failed). State machine in `internal/core/resource.go`
- Provisioner interface in `internal/jobs/provisioner.go` — `Provision()`, `Delete()`, `HealthCheck()` return `ProvisionResult`
- Policy engine: `internal/core/policy.go` evaluates rules (eq/neq/in/nin/lt/gt). Effects: allow, deny, require_approval. Deny wins.
- Quota checks: `internal/core/quota.go` — per-project or global, wildcard `*` resource type

**Rules:**
- Always add AGPL v3 license header to new Go files
- Handler functions live in their own file under `internal/api/` (e.g., `resources.go`, `approvals.go`)
- Business logic belongs in `internal/core/`, not in handlers
- New API routes registered in `router.go` under the `/api/v1` group
- Use `respondJSON()` and `respondError()` from `internal/api/respond.go` — never write raw HTTP responses

## database

Database and query specialist. Works in `internal/db/`, SQL migrations, sqlc config.

**Knows:**
- 13-table PostgreSQL 18 schema in `internal/db/migrations/00001_initial_schema.sql`
- Tables: users, teams, team_members, projects, memberships, resource_types, resources, provisioning_events, policies, policy_rules, quotas, approval_requests, audit_entries
- sqlc generates Go code from SQL in `internal/db/queries/*.sql` → `internal/db/*.sql.go`
- Config: `sqlc.yaml` at repo root, engine postgresql, package `db`, pgx/v5
- Pool init: `internal/db/pool.go` — `NewPool(ctx, databaseURL) (*pgxpool.Pool, error)`
- Migrations use goose format: `-- +goose Up` / `-- +goose Down`
- All IDs are `TEXT` with `gen_random_uuid()::text` defaults

**Rules:**
- Write SQL queries in `internal/db/queries/<table>.sql`, then run `make generate`
- Never hand-edit `internal/db/*.sql.go` — these are sqlc-generated
- New migrations go in `internal/db/migrations/` numbered sequentially (e.g., `00002_add_feature.sql`)
- Always include both `-- +goose Up` and `-- +goose Down` sections
- Use `sqlc.narg()` for nullable parameters, `:one`/`:many`/`:exec` annotations

## frontend

React/TypeScript frontend specialist. Works in `web/`.

**Knows:**
- Vite 7 + React 19 + TypeScript 5.9
- React Router v6 in `web/src/App.tsx` — Layout wrapper with sidebar nav
- API client in `web/src/api.ts` — talks to `http://localhost:8080/api/v1`
- Pages: CatalogPage, ResourcesPage, ResourceDetailPage, ProvisionPage, ApprovalListPage, ApprovalDetailPage
- Layout with sidebar nav in `web/src/Layout.tsx`
- Vite dev server on port 5173, proxied to Go backend on 8080

**Rules:**
- New pages go in `web/src/pages/` and get a route in `App.tsx`
- Shared components go in `web/src/components/`
- Use TypeScript strict mode — no `any`, no unused vars
- API types should mirror the Go JSON responses

## devops

Infrastructure and deployment specialist. Works in `deploy/`, `docker-compose.yml`, `.github/workflows/`, `Makefile`.

**Knows:**
- Docker Compose: PostgreSQL 18 + Valkey 8 for local dev
- Helm chart in `deploy/helm/portway/` — two profiles: standalone (Bitnami PostgreSQL + official valkey-io chart) and external (managed services)
- CI: `.github/workflows/ci.yml` — Go test + build + golangci-lint
- Makefile targets: build, run, run-worker, test, lint, generate, migrate-up, migrate-down, dev, clean, build-cli
- Three binaries: `cmd/portway` (API), `cmd/worker` (Asynq), `cmd/portway-cli` (CLI)

**Rules:**
- Helm values use `valkey.enabled`/`postgresql.enabled` conditions for subchart toggling
- Official Valkey chart at `https://valkey.io/valkey-helm/` — uses `dataStorage` not `primary.persistence`
- Keep CI fast — only add steps that catch real issues
