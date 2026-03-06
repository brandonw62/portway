# Portway

A lightweight open-source Internal Developer Portal (IDP) — a Backstage
alternative built for solo developers and small engineering teams.

## Stack

| Layer       | Technology                                      |
|-------------|-------------------------------------------------|
| Backend     | Go 1.26, Chi, sqlc, Asynq, zerolog              |
| Frontend    | React + TypeScript (Vite)                       |
| Database    | PostgreSQL 18                                   |
| Queue       | Valkey 8 + Asynq                                |
| Deploy      | Docker Compose (dev), Helm (production)         |

## Quick Start (Development)

### Prerequisites

- Go 1.26+
- Docker + Docker Compose
- Node 20+ and npm (for the frontend)
- [sqlc](https://sqlc.dev) — `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- [goose](https://github.com/pressly/goose) — `go install github.com/pressly/goose/v3/cmd/goose@latest`
- [golangci-lint](https://golangci-lint.run) — follow install guide (do not use `go install`)

### Setup

```bash
# 1. Start backing services (PostgreSQL 18 + Valkey 8)
make dev

# 2. Copy and review the environment template
cp .env.example .env

# 3. Apply database migrations
source .env && make migrate-up

# 4. Build and start the API server
make run

# 5. In another terminal, start the worker
make run-worker

# 6. Set up the frontend (first time only)
npm create vite@latest web -- --template react-ts
cd web && npm install && npm run dev
```

Confirm the server is running:

```bash
curl http://localhost:8080/healthz
# {"status":"ok"}
```

## Makefile Targets

| Target             | Description                                        |
|--------------------|----------------------------------------------------|
| `make build`       | Compile both `bin/portway` and `bin/worker`        |
| `make run`         | Build and start the API server                     |
| `make run-worker`  | Build and start the Asynq background worker        |
| `make test`        | Run all Go tests with race detector                |
| `make lint`        | Run golangci-lint                                  |
| `make generate`    | Run sqlc + go generate                             |
| `make migrate-up`  | Apply all pending DB migrations via goose          |
| `make migrate-down`| Roll back the last migration via goose             |
| `make dev`         | Start postgres + valkey via Docker Compose         |
| `make clean`       | Remove compiled binaries                           |

## Helm Deployment

Two profiles are provided under `deploy/helm/portway/`.

### Standalone (in-cluster PostgreSQL + Valkey)

```bash
helm install portway ./deploy/helm/portway -f deploy/helm/portway/values.yaml
```

### External (managed RDS + hosted Valkey)

```bash
helm install portway ./deploy/helm/portway \
  -f deploy/helm/portway/values.yaml \
  -f deploy/helm/portway/values-external.yaml \
  --set externalDatabase.url="postgres://..." \
  --set externalValkey.url="redis://..."
```

## Architecture

```
cmd/
  portway/   — API server entrypoint
  worker/    — Asynq background worker entrypoint
internal/
  api/       — HTTP handlers, middleware, router
  config/    — Env-var configuration
  core/      — Domain types and sentinel errors
  db/        — pgx pool, sqlc-generated code, migrations
  integrations/ — GitHub App, AWS, PagerDuty clients
  jobs/      — Asynq task definitions and enqueue helpers
web/         — React + TypeScript frontend (Vite)
deploy/helm/ — Helm chart (standalone + external profiles)
```

## License

Portway is licensed under the [GNU Affero General Public License v3.0](LICENSE)
with a dual commercial license available for organizations that require it.
Contact licensing@portway.dev for details.
