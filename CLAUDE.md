# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Build all binaries (server, worker, CLI)
make build

# Run API server (reads .env)
make run

# Run background worker
make run-worker

# Start backing services (PostgreSQL 18 + Valkey 8)
make dev

# Run all tests with race detector
make test

# Run a single test
go test -run TestFunctionName ./internal/package/...

# Generate sqlc code
make generate

# Apply / roll back DB migrations (requires DATABASE_URL in env)
make migrate-up
make migrate-down

# Build CLI only
make build-cli

# Frontend dev server
cd web && npm run dev
```

## Architecture

Three binaries, one module (`github.com/portway/portway`):

- **`cmd/portway`** — HTTP API server. Wires `config → db pool → Chi router → http.Server` with graceful shutdown on SIGINT/SIGTERM.
- **`cmd/worker`** — Asynq background worker. Wires `config → db pool → asynq.Server → task mux`. Queues: `critical` (weight 6), `default` (3), `low` (1).
- **`cmd/portway-cli`** — Developer CLI for provisioning resources from the terminal. Build: `make build-cli`. See [CLI usage](#cli-portway-cli) below.

### Internal packages

| Package | Role |
|---|---|
| `internal/config` | Single `Config` struct parsed from env vars via `caarlos0/env`. All env var additions go here first. |
| `internal/api` | Chi router + middleware (zerolog hlog access logs, request ID, recoverer). New routes registered in `router.go`. |
| `internal/core` | Sentinel errors (`ErrNotFound`, `ErrConflict`, etc.) that handlers map to HTTP status codes. Domain types live here as the project grows. |
| `internal/db` | pgx/v5 pool init. sqlc-generated query code goes here (run `make generate`). Migrations in `internal/db/migrations/` using goose format (`-- +goose Up` / `-- +goose Down`). |
| `internal/jobs` | Asynq task type constants and typed enqueue helpers. `TypeGitHubSync` is the first defined task. |
| `internal/integrations` | External service clients (GitHub App, AWS, PagerDuty) — empty at scaffold, to be built out. |

### Key conventions

- **Error handling**: return sentinel errors from `internal/core`; map them to HTTP status in handlers. Never return raw HTTP errors from business logic.
- **GitHub App auth**: use `bradleyfalzon/ghinstallation/v2` + `google/go-github/v68` for installation tokens; cache in Valkey with 55-min TTL (tokens expire at 60 min). Add to `go.mod` when `internal/integrations/github/` is built.
- **Config**: `ENVIRONMENT=development` enables console-formatted zerolog output; any other value uses JSON.
- **Database queries**: write SQL in `internal/db/queries/`, run `make generate` to produce Go code via sqlc. Schema changes go in `internal/db/migrations/` as goose-formatted `.sql` files.
- **Task registration**: new Asynq task types get a constant in `internal/jobs/jobs.go` and a typed enqueue method; handler registration goes in `cmd/worker/main.go`.

### Frontend (`web/`)

Vite 7 + React 19 + TypeScript 5.9. Not yet wired to the API. Dev server: `cd web && npm run dev` (port 5173). Build: `cd web && npm run build`.

### CLI (`portway-cli`)

Developer CLI for resource provisioning. Configure via environment variables:

- `PORTWAY_API_URL` — API server URL (default: `http://localhost:8080`)
- `PORTWAY_TOKEN` — Authentication bearer token

```bash
# List the resource catalog
portway-cli catalog

# Provision a resource
portway-cli provision <resource-type-id> --name my-db --project <project-id> --spec instance_class=db.t3.micro --spec allocated_gb=50

# List resources
portway-cli resources --project <project-id>
portway-cli resources --status ready

# Get resource detail
portway-cli status <resource-id>

# Delete a resource
portway-cli delete <resource-id>

# List pending approvals
portway-cli approvals

# Approve a request
portway-cli approve <approval-id> --comment "Looks good"
```

### Deployment

- **Local dev**: `docker compose up -d` (postgres + valkey), then `make run`.
- **Helm**: `deploy/helm/portway/` has two value files — `values.yaml` (standalone: in-cluster PostgreSQL via Bitnami + Valkey via official `valkey-io/valkey-helm`) and `values-external.yaml` (external managed services overlay).
