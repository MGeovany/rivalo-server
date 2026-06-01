// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
)

// Config holds the runtime configuration for the server.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string
	// DatabaseURL is the PostgreSQL connection string (Supabase). Optional in
	// local development: when empty, database-backed features are disabled and
	// the server still serves stateless endpoints such as /health.
	DatabaseURL string
	// SupabaseJWTSecret is the secret used to validate Supabase Auth JWTs.
	// Optional until authenticated endpoints are wired up.
	SupabaseJWTSecret string
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() Config {
	return Config{
		Port:              getenv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		SupabaseJWTSecret: os.Getenv("SUPABASE_JWT_SECRET"),
	}
}

// getenv returns the value of the environment variable named by key, or
// fallback when the variable is unset or empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
