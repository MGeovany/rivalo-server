package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/MGeovany/rivalo-server/internal/logger"
)

// contextKey is an unexported type for context keys defined in this package.
type contextKey string

const userIDKey contextKey = "userID"

// requireAuth wraps a handler, rejecting requests without a valid Supabase JWT.
// On success the authenticated user id is stored in the request context.
func (d Deps) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Verifier.Configured() {
			logger.Warn("auth_rejected", "reason", "not_configured", "path", r.URL.Path)
			writeError(w, http.StatusServiceUnavailable, "authentication is not configured")
			return
		}

		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			logger.Warn("auth_rejected", "reason", "missing_token", "path", r.URL.Path)
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		userID, err := d.Verifier.Verify(token)
		if err != nil {
			logger.Warn("auth_rejected", "reason", "invalid_token", "path", r.URL.Path)
			writeError(w, http.StatusUnauthorized, "invalid token")
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
