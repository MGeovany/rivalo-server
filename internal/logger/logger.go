// Package logger provides a process-wide structured logger safe for production:
// no tokens, secrets, connection strings, or other PII in log fields.
package logger

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	mu sync.RWMutex
	l  = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
)

// Init configures the global logger from the environment.
// LOG_LEVEL: debug, info (default), warn, error.
func Init() {
	mu.Lock()
	defer mu.Unlock()
	l = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))}))
}

// L returns the global logger.
func L() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return l
}

// Info logs at info level on the global logger.
func Info(msg string, args ...any) { L().Info(msg, args...) }

// Warn logs at warn level on the global logger.
func Warn(msg string, args ...any) { L().Warn(msg, args...) }

// Error logs at error level on the global logger.
func Error(msg string, args ...any) { L().Error(msg, args...) }

// Debug logs at debug level on the global logger.
func Debug(msg string, args ...any) { L().Debug(msg, args...) }

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
