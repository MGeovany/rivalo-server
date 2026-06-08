package profile

import (
	"context"
	"time"

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

const profileColumns = `id, display_name, preferred_position, height_cm, weight_kg, birth_year, birth_date, created_at, updated_at`

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
		insert into public.profiles (id, display_name, preferred_position, height_cm, weight_kg, birth_year, birth_date)
		values ($1, $2, $3, $4, $5, $6, $7)
		on conflict (id) do update set
			display_name       = excluded.display_name,
			preferred_position = excluded.preferred_position,
			height_cm          = excluded.height_cm,
			weight_kg          = excluded.weight_kg,
			birth_year         = excluded.birth_year,
			birth_date         = excluded.birth_date,
			updated_at         = now()
		returning ` + profileColumns

	return scanProfile(s.pool.QueryRow(ctx, query, id, u.DisplayName, u.PreferredPosition, u.HeightCM, u.WeightKG, u.BirthYear, u.BirthDate))
}

// row is the subset of pgx.Row used here, satisfied by pgxpool query results.
type row interface {
	Scan(dest ...any) error
}

func scanProfile(r row) (Profile, error) {
	var p Profile
	var birthDate *time.Time
	err := r.Scan(
		&p.ID,
		&p.DisplayName,
		&p.PreferredPosition,
		&p.HeightCM,
		&p.WeightKG,
		&p.BirthYear,
		&birthDate,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if birthDate != nil {
		s := birthDate.Format("2006-01-02")
		p.BirthDate = &s
	}
	return p, err
}
