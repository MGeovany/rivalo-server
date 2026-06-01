// Package db manages the PostgreSQL (Supabase) connection pool.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a connection pool to PostgreSQL.
type DB struct {
	Pool *pgxpool.Pool
}

// Connect opens a connection pool to the database at url and verifies it with a
// ping. The caller is responsible for calling Close.
func Connect(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Ping verifies the database is reachable. It satisfies the httpapi.Pinger
// interface used by the health endpoint.
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Close releases the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}
