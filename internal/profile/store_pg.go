package profile

import (
	"context"

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

const profileColumns = `id, display_name, preferred_position, height_cm, weight_kg, birth_year, created_at, updated_at`

func (s *PostgresStore) GetOrCreate(ctx context.Context, id string) (Profile, error) {
	// Insert a default row on first access, then return the current row whether
	// it was just created or already existed.
	const query = `
		with ins as (
			insert into public.profiles (id) values ($1)
			on conflict (id) do nothing
			returning ` + profileColumns + `
		)
		select ` + profileColumns + ` from ins
		union all
		select ` + profileColumns + ` from public.profiles where id = $1
		limit 1`

	return scanProfile(s.pool.QueryRow(ctx, query, id))
}

func (s *PostgresStore) Update(ctx context.Context, id string, u Update) (Profile, error) {
	const query = `
		insert into public.profiles (id, display_name, preferred_position, height_cm, weight_kg, birth_year)
		values ($1, $2, $3, $4, $5, $6)
		on conflict (id) do update set
			display_name       = excluded.display_name,
			preferred_position = excluded.preferred_position,
			height_cm          = excluded.height_cm,
			weight_kg          = excluded.weight_kg,
			birth_year         = excluded.birth_year,
			updated_at         = now()
		returning ` + profileColumns

	return scanProfile(s.pool.QueryRow(ctx, query, id, u.DisplayName, u.PreferredPosition, u.HeightCM, u.WeightKG, u.BirthYear))
}

// row is the subset of pgx.Row used here, satisfied by pgxpool query results.
type row interface {
	Scan(dest ...any) error
}

func scanProfile(r row) (Profile, error) {
	var p Profile
	err := r.Scan(
		&p.ID,
		&p.DisplayName,
		&p.PreferredPosition,
		&p.HeightCM,
		&p.WeightKG,
		&p.BirthYear,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	return p, err
}
