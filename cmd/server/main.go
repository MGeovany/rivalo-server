// Command server starts the Rivalo HTTP API.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MGeovany/rivalo-server/internal/config"
	"github.com/MGeovany/rivalo-server/internal/db"
	"github.com/MGeovany/rivalo-server/internal/httpapi"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func run() error {
	cfg := config.Load()

	// The database is optional in local development. When DATABASE_URL is set we
	// connect and expose it to handlers; otherwise the API still serves
	// stateless endpoints such as /health.
	var pinger httpapi.Pinger
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		database, err := db.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer database.Close()
		pinger = database
		log.Println("connected to database")
	} else {
		log.Println("DATABASE_URL not set; running without database")
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(pinger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Run the server until an interrupt signal arrives, then shut down cleanly.
	shutdownErr := make(chan error, 1)
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		shutdownErr <- srv.Shutdown(ctx)
	}()

	log.Printf("listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return <-shutdownErr
}
