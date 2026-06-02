package logger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/MGeovany/rivalo-server/internal/auth"
	"github.com/jackc/pgx/v5/pgconn"
)

// Ref returns an opaque, stable reference for a user or resource id (never the raw value).
func Ref(key, id string) slog.Attr {
	if id == "" {
		return slog.String(key, "")
	}
	sum := sha256.Sum256([]byte(id))
	return slog.String(key, hex.EncodeToString(sum[:4]))
}

// SafeErr logs a coarse but actionable error classification without secrets, URLs, or PII.
func SafeErr(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "unknown")
	}
	if errors.Is(err, auth.ErrNotConfigured) {
		return slog.String("error", "auth_not_configured")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return slog.Group("error",
			slog.String("kind", "postgres"),
			slog.String("code", pgErr.Code),
			slog.String("detail", pgErr.Message),
		)
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return slog.Group("error", slog.String("kind", "json_syntax"))
	}

	msg := err.Error()
	if sensitiveErrorText(msg) {
		return slog.Group("error", slog.String("kind", fmt.Sprintf("%T", err)))
	}

	return slog.Group("error",
		slog.String("kind", fmt.Sprintf("%T", err)),
		slog.String("message", msg),
	)
}

func sensitiveErrorText(msg string) bool {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "password") || strings.Contains(lower, "secret") {
		return true
	}
	if strings.Contains(msg, "://") {
		return true
	}
	return false
}
