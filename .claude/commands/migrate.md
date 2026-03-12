Create a new goose SQL migration file in `internal/db/migrations/`.

1. Find the highest-numbered migration file in `internal/db/migrations/` and increment by 1
2. Create the new file with the pattern `NNNNN_$ARGUMENTS.sql` (e.g., `00002_add_deployments.sql`)
3. Include both `-- +goose Up` and `-- +goose Down` sections
4. Write the SQL based on what the user describes in $ARGUMENTS
5. After creating the migration, remind the user to run `make migrate-up` and `make generate` if new tables/columns affect sqlc queries
