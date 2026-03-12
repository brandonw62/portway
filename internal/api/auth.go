package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/oauth2"

	"github.com/portway/portway/internal/db"
)

// AuthConfig holds the parameters needed to set up OIDC auth.
type AuthConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Environment  string
	Queries      *db.Queries
}

// OIDCEnabled returns true when all required OIDC settings are present.
func (c AuthConfig) OIDCEnabled() bool {
	return c.IssuerURL != "" && c.ClientID != "" && c.ClientSecret != "" && c.RedirectURL != ""
}

// authHandler holds OIDC provider state for the auth endpoints.
type authHandler struct {
	provider    *oidc.Provider
	verifier    *oidc.IDTokenVerifier
	oauth2Cfg   oauth2.Config
	queries     *db.Queries
	environment string
}

// newAuthHandler initializes the OIDC provider and returns the handler.
// Returns nil if OIDC is not configured.
func newAuthHandler(ctx context.Context, cfg AuthConfig) (*authHandler, error) {
	if !cfg.OIDCEnabled() {
		return nil, nil
	}

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	return &authHandler{
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		oauth2Cfg: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		queries:     cfg.Queries,
		environment: cfg.Environment,
	}, nil
}

// HandleLogin redirects the user to the OIDC provider's authorization endpoint.
func (h *authHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomState()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate state")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   300,
	})

	http.Redirect(w, r, h.oauth2Cfg.AuthCodeURL(state), http.StatusFound)
}

// HandleCallback exchanges the authorization code for tokens and sets a session.
func (h *authHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Validate state parameter.
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		respondError(w, http.StatusBadRequest, "invalid oauth state")
		return
	}

	// Clear the state cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	oauth2Token, err := h.oauth2Cfg.Exchange(r.Context(), code)
	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("oauth2 token exchange failed")
		respondError(w, http.StatusUnauthorized, "token exchange failed")
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		respondError(w, http.StatusInternalServerError, "no id_token in response")
		return
	}

	idToken, err := h.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid id_token")
		return
	}

	// Extract claims and JIT-provision the user.
	user, err := h.jitProvision(r.Context(), idToken)
	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("JIT user provisioning failed")
		respondError(w, http.StatusInternalServerError, "user provisioning failed")
		return
	}

	// Set the bearer token cookie for the frontend.
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    rawIDToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   3600,
	})

	_ = user // provisioned; token is set via cookie
	// Redirect to the frontend app.
	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleMe returns the currently authenticated user.
func (h *authHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	respondJSON(w, http.StatusOK, user)
}

// AuthMiddleware validates the Bearer token or auth_token cookie on each request.
// In development mode with no OIDC configured, it falls back to X-User-Id.
func (h *authHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawToken := extractBearerToken(r)

		if rawToken == "" {
			respondError(w, http.StatusUnauthorized, "missing authorization token")
			return
		}

		idToken, err := h.verifier.Verify(r.Context(), rawToken)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		user, err := h.jitProvision(r.Context(), idToken)
		if err != nil {
			hlog.FromRequest(r).Error().Err(err).Msg("failed to resolve user from token")
			respondError(w, http.StatusInternalServerError, "user lookup failed")
			return
		}

		ctx := ContextWithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// devAuthMiddleware is used in development mode when OIDC is not configured.
// It reads user identity from the X-User-Id header for local testing.
func devAuthMiddleware(queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-Id")
			if userID == "" {
				respondError(w, http.StatusUnauthorized, "X-User-Id header required in development mode")
				return
			}

			user, err := queries.GetUser(r.Context(), userID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					respondError(w, http.StatusUnauthorized, "unknown user")
					return
				}
				respondError(w, http.StatusInternalServerError, "user lookup failed")
				return
			}

			ctx := ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type oidcClaims struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// jitProvision looks up or creates a user based on OIDC token claims.
func (h *authHandler) jitProvision(ctx context.Context, idToken *oidc.IDToken) (db.User, error) {
	var claims oidcClaims
	if err := idToken.Claims(&claims); err != nil {
		return db.User{}, err
	}

	issuerSub := idToken.Issuer + "|" + idToken.Subject

	// Try to find an existing user by issuer+subject.
	user, err := h.queries.GetUserByIssuerSub(ctx, issuerSub)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, err
	}

	// No existing user — create one.
	name := claims.Name
	if name == "" {
		name = claims.Email
	}

	user, err = h.queries.CreateUser(ctx, db.CreateUserParams{
		Email:     claims.Email,
		Name:      name,
		AvatarUrl: claims.Picture,
		IssuerSub: issuerSub,
	})
	if err != nil {
		// Handle race condition: another request may have created the user.
		if strings.Contains(err.Error(), "duplicate key") {
			return h.queries.GetUserByIssuerSub(ctx, issuerSub)
		}
		return db.User{}, err
	}

	return user, nil
}

// extractBearerToken gets the token from Authorization header or auth_token cookie.
func extractBearerToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	if cookie, err := r.Cookie("auth_token"); err == nil {
		return cookie.Value
	}
	return ""
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// meResponse is returned by the /auth/me endpoint.
type meResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// devHandleMe returns the user from context for development mode.
func devHandleMe(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-Id")
		if userID == "" {
			respondError(w, http.StatusUnauthorized, "X-User-Id header required")
			return
		}
		user, err := queries.GetUser(r.Context(), userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondError(w, http.StatusUnauthorized, "unknown user")
				return
			}
			respondError(w, http.StatusInternalServerError, "user lookup failed")
			return
		}
		respondJSON(w, http.StatusOK, meResponse{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			AvatarURL: user.AvatarUrl,
		})
	}
}
