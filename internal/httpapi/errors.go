package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/MGeovany/rivalo-server/internal/logger"
)

// logAndWriteError records the failure (with optional err) then writes the JSON error body.
// Use on every handler error path so failures are visible in server logs.
func logAndWriteError(w http.ResponseWriter, status int, clientMsg, logMsg string, err error, attrs ...any) {
	args := append([]any{}, attrs...)
	args = append(args, slog.Int("status", status))
	if err != nil {
		args = append(args, logger.SafeErr(err))
	}
	if status >= http.StatusInternalServerError {
		logger.Error(logMsg, args...)
	} else {
		logger.Warn(logMsg, args...)
	}
	writeError(w, status, clientMsg)
}
