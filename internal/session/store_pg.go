package session

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is a PostgreSQL-backed Store.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore builds a PostgresStore over the given pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

const sessionColumns = `id, user_id, started_at, ended_at, duration_s, distance_m,
	hr_avg, hr_max, speed_max_kmh, sprints, intensity, calories_kcal, source, created_at`

func (s *PostgresStore) Create(ctx context.Context, userID string, n New) (Session, error) {
	const query = `
		insert into public.sessions
			(user_id, started_at, ended_at, duration_s, distance_m,
			 hr_avg, hr_max, speed_max_kmh, sprints, intensity, calories_kcal, source)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		returning ` + sessionColumns

	return scanSession(s.pool.QueryRow(ctx, query,
		userID, n.StartedAt, n.EndedAt, n.DurationS, n.DistanceM,
		n.HRAvg, n.HRMax, n.SpeedMaxKMH, n.Sprints, n.Intensity, n.CaloriesKcal, n.Source,
	))
}

func (s *PostgresStore) List(ctx context.Context, userID string) ([]Session, error) {
	const query = `
		select ` + sessionColumns + `
		from public.sessions
		where user_id = $1
		order by started_at desc`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]Session, 0)
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *PostgresStore) Get(ctx context.Context, userID, id string) (Session, error) {
	const query = `
		select ` + sessionColumns + `
		from public.sessions
		where id = $1 and user_id = $2`

	sess, err := scanSession(s.pool.QueryRow(ctx, query, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	return sess, err
}

// scanRow is satisfied by both pgx.Row and pgx.Rows.
type scanRow interface {
	Scan(dest ...any) error
}

func scanSession(r scanRow) (Session, error) {
	var s Session
	err := r.Scan(
		&s.ID, &s.UserID, &s.StartedAt, &s.EndedAt, &s.DurationS, &s.DistanceM,
		&s.HRAvg, &s.HRMax, &s.SpeedMaxKMH, &s.Sprints, &s.Intensity, &s.CaloriesKcal,
		&s.Source, &s.CreatedAt,
	)
	return s, err
}
