# Portway — top-level Makefile
# All targets assume they are run from the repository root.

BINARY_SERVER := bin/portway
BINARY_WORKER := bin/worker
GO            := go
GOFLAGS       :=

.PHONY: all build build-server build-worker run run-worker test lint \
        generate migrate-up migrate-down dev clean

## all: build both binaries (default target)
all: build

## build: compile both the API server and background worker
build: build-server build-worker

## build-server: compile the API server binary to bin/portway
build-server:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -o $(BINARY_SERVER) ./cmd/portway

## build-worker: compile the background worker binary to bin/worker
build-worker:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -o $(BINARY_WORKER) ./cmd/worker

## run: start the API server (reads .env if present)
run: build-server
	@[ -f .env ] && export $$(grep -v '^#' .env | xargs) ; \
	./$(BINARY_SERVER)

## run-worker: start the background worker
run-worker: build-worker
	@[ -f .env ] && export $$(grep -v '^#' .env | xargs) ; \
	./$(BINARY_WORKER)

## test: run all Go tests with the race detector
test:
	$(GO) test -race -count=1 ./...

## lint: run golangci-lint (must be installed separately)
lint:
	golangci-lint run ./...

## generate: run sqlc code generation and any //go:generate directives
generate:
	sqlc generate
	$(GO) generate ./...

## migrate-up: apply all pending migrations using goose
## Requires: go install github.com/pressly/goose/v3/cmd/goose@latest
migrate-up:
	goose -dir internal/db/migrations postgres "$(DATABASE_URL)" up

## migrate-down: roll back the most recent migration using goose
migrate-down:
	goose -dir internal/db/migrations postgres "$(DATABASE_URL)" down

## dev: start postgres and valkey via docker compose in detached mode
dev:
	docker compose up -d

## clean: remove compiled binaries
clean:
	rm -rf bin/
