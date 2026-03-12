Create a new sqlc query or add queries to an existing file in `internal/db/queries/`.

1. Determine which table the query targets from $ARGUMENTS
2. Check if `internal/db/queries/<table>.sql` already exists — append to it if so, create it if not
3. Write the SQL query with proper sqlc annotations (`:one`, `:many`, `:exec`, `:execrows`)
4. Use `sqlc.narg()` for nullable filter parameters
5. Run `make generate` to produce the Go code
6. Verify the generated code compiles: `go build ./internal/db/...`
7. Show the generated Go function signature to the user
