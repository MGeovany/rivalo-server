package httpapi

import (
	"context"
	"net/http"
	"strings"


)

// contextKey is an unexported type for context keys defined in this package.
type contextKey string

const userIDKey contextKey = "userID"

// requireAuth wraps a handler, rejecting requests without a valid Supabase JWT.
// On success the authenticated user id is stored in the request context.
func (d Deps) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Verifier.Configured() {
			logAndWriteError(w, http.StatusServiceUnavailable, "authentication is not configured", "auth_rejected", nil, "reason", "not_configured", "path", r.URL.Path)
			return
		}

		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			logAndWriteError(w, http.StatusUnauthorized, "missing bearer token", "auth_rejected", nil, "reason", "missing_token", "path", r.URL.Path)
			return
		}

		userID, err := d.Verifier.Verify(token)
		if err != nil {
			logAndWriteError(w, http.StatusUnauthorized, "invalid token", "auth_rejected", err, "reason", "invalid_token", "path", r.URL.Path)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header.
func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}

// userID returns the authenticated user id stored by requireAuth.
func userID(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}
