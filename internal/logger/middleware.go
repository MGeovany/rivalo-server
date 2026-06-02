package logger

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// RequestLog wraps an HTTP handler with access-style logging. Paths under /docs and
// GET /health are skipped to avoid noise. Query strings are never logged.
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if skipRequestLog(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		Info("http_request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

func skipRequestLog(method, path string) bool {
	if strings.HasPrefix(path, "/docs") {
		return true
	}
	return method == http.MethodGet && path == "/health"
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
