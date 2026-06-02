// Command server starts the Rivalo HTTP API.
//
//	@title			Rivalo API
//	@version		0.1.0
//	@description	Backend API for Rivalo (profiles, sessions, health).
//	@host			localhost:8080
//	@BasePath		/
//	@schemes		http
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MGeovany/rivalo-server/internal/auth"
	"github.com/MGeovany/rivalo-server/internal/config"
	"github.com/MGeovany/rivalo-server/internal/db"
	"github.com/MGeovany/rivalo-server/internal/httpapi"
	"github.com/MGeovany/rivalo-server/internal/logger"
	"github.com/MGeovany/rivalo-server/internal/profile"
	"github.com/MGeovany/rivalo-server/internal/session"
)

func main() {
	logger.Init()
	if err := run(); err != nil {
		logger.Error("server_exit", logger.SafeErr(err))
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	// The database is optional in local development. When DATABASE_URL is set we
	// connect and expose it to handlers; otherwise the API still serves
	// stateless endpoints such as /health.
	var pinger httpapi.Pinger
	var profiles profile.Store
	var sessions session.Store
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		database, err := db.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			logger.Error("database_connect_failed", logger.SafeErr(err))
			return err
		}
		defer database.Close()
		pinger = database
		profiles = profile.NewPostgresStore(database.Pool)
		sessions = session.NewPostgresStore(database.Pool)
		logger.Info("database_ready")
	} else {
		logger.Warn("database_disabled")
	}

	verifier := buildVerifier(cfg)
	logger.Info("auth_ready", slog.Bool("configured", verifier.Configured()))

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewRouter(httpapi.Deps{
			DB:       pinger,
			Profiles: profiles,
			Sessions: sessions,
			Verifier: verifier,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Run the server until an interrupt signal arrives, then shut down cleanly.
	shutdownErr := make(chan error, 1)
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		logger.Info("shutdown_signal")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		shutdownErr <- srv.Shutdown(ctx)
	}()

	logger.Info("server_listening", slog.String("port", cfg.Port))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("server_stopped")
	return <-shutdownErr
}

// buildVerifier picks the JWT verification strategy from config: the project's
// JWKS (asymmetric ES256, preferred) when SUPABASE_URL is set, otherwise the
// legacy HS256 shared secret. A failure to load the JWKS leaves the verifier
// unconfigured so the rest of the API still serves.
func buildVerifier(cfg config.Config) auth.Verifier {
	if cfg.SupabaseURL != "" {
		jwksURL := strings.TrimRight(cfg.SupabaseURL, "/") + "/auth/v1/.well-known/jwks.json"
		verifier, err := auth.NewJWKSVerifier(jwksURL)
		if err != nil {
			logger.Error("auth_jwks_failed", logger.SafeErr(err))
			return auth.Verifier{}
		}
		return verifier
	}
	return auth.NewVerifier(cfg.SupabaseJWTSecret)
}
