package api

import (
	"context"

	"github.com/portway/portway/internal/db"
)

type contextKey int

const userContextKey contextKey = iota

// ContextWithUser returns a new context carrying the authenticated user.
func ContextWithUser(ctx context.Context, u db.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

// UserFromContext extracts the authenticated user from the context.
// Returns the user and true if present, or a zero value and false if not.
func UserFromContext(ctx context.Context) (db.User, bool) {
	u, ok := ctx.Value(userContextKey).(db.User)
	return u, ok
}

// MustUserFromContext extracts the authenticated user from the context.
// It panics if no user is present (use only behind auth middleware).
func MustUserFromContext(ctx context.Context) db.User {
	u, ok := UserFromContext(ctx)
	if !ok {
		panic("api: MustUserFromContext called without authenticated user in context")
	}
	return u
}
