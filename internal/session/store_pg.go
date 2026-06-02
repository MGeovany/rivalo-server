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

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)

	sess, err := scanSession(tx.QueryRow(ctx, query,
		userID, n.StartedAt, n.EndedAt, n.DurationS, n.DistanceM,
		n.HRAvg, n.HRMax, n.SpeedMaxKMH, n.Sprints, n.Intensity, n.CaloriesKcal, n.Source,
	))
	if err != nil {
		return Session{}, err
	}

	if len(n.Samples) > 0 {
		rows := make([][]any, len(n.Samples))
		for i, smp := range n.Samples {
			rows[i] = []any{sess.ID, smp.TOffsetS, smp.HR, smp.SpeedKMH}
		}
		_, err = tx.CopyFrom(ctx,
			pgx.Identifier{"public", "session_samples"},
			[]string{"session_id", "t_offset_s", "hr", "speed_kmh"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return Session{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	sess.Samples = n.Samples
	return sess, nil
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
	if err != nil {
		return Session{}, err
	}

	samples, err := s.loadSamples(ctx, sess.ID)
	if err != nil {
		return Session{}, err
	}
	sess.Samples = samples
	return sess, nil
}

func (s *PostgresStore) loadSamples(ctx context.Context, sessionID string) ([]Sample, error) {
	const query = `
		select t_offset_s, hr, speed_kmh
		from public.session_samples
		where session_id = $1
		order by t_offset_s`

	rows, err := s.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	samples := make([]Sample, 0)
	for rows.Next() {
		var smp Sample
		if err := rows.Scan(&smp.TOffsetS, &smp.HR, &smp.SpeedKMH); err != nil {
			return nil, err
		}
		samples = append(samples, smp)
	}
	return samples, rows.Err()
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
