package logger

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"

	"github.com/MGeovany/rivalo-server/internal/auth"
)

// Ref returns an opaque, stable reference for a user or resource id (never the raw value).
func Ref(key, id string) slog.Attr {
	if id == "" {
		return slog.String(key, "")
	}
	sum := sha256.Sum256([]byte(id))
	return slog.String(key, hex.EncodeToString(sum[:4]))
}

// SafeErr logs only a coarse error classification, not wrapped driver messages that
// may contain hosts, URLs, or credentials.
func SafeErr(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "unknown")
	}
	if errors.Is(err, auth.ErrNotConfigured) {
		return slog.String("error", "auth_not_configured")
	}
	return slog.String("error", "failed")
}
