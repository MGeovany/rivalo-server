package badge

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Earned(ctx context.Context, userID string) (map[string]time.Time, error) {
	const query = `select badge_key, earned_at from public.user_badges where user_id = $1`
	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	earned := map[string]time.Time{}
	for rows.Next() {
		var key string
		var at time.Time
		if err := rows.Scan(&key, &at); err != nil {
			return nil, err
		}
		earned[key] = at
	}
	return earned, rows.Err()
}

func (s *PostgresStore) Grant(ctx context.Context, userID, key string, at time.Time) error {
	const query = `
		insert into public.user_badges (user_id, badge_key, earned_at)
		values ($1, $2, $3)
		on conflict (user_id, badge_key) do nothing`
	_, err := s.pool.Exec(ctx, query, userID, key, at)
	return err
}
