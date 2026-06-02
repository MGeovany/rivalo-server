package pitch

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

const pitchColumns = `id, user_id, name, latitude, longitude, type, surface,
	length_m, width_m, measurement_method, indoor, notes,
	created_at, updated_at`

func (s *PostgresStore) Create(ctx context.Context, userID string, n NewPitch) (Pitch, error) {
	const query = `
		insert into public.pitches
			(user_id, name, latitude, longitude, type, surface, length_m, width_m, measurement_method,
			 indoor, notes)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		returning ` + pitchColumns
	return scanPitch(s.pool.QueryRow(ctx, query,
		userID, n.Name, n.Latitude, n.Longitude, n.Type, n.Surface,
		n.LengthM, n.WidthM, n.MeasurementMethod, n.Indoor, n.Notes,
	))
}

func (s *PostgresStore) Get(ctx context.Context, userID, id string) (Pitch, error) {
	const query = `
		select ` + pitchColumns + `
		from public.pitches
		where id = $1 and user_id = $2`
	p, err := scanPitch(s.pool.QueryRow(ctx, query, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Pitch{}, ErrNotFound
	}
	return p, err
}

func (s *PostgresStore) List(ctx context.Context, userID string) ([]Pitch, error) {
	const query = `
		select ` + pitchColumns + `
		from public.pitches
		where user_id = $1
		order by name`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pitches := make([]Pitch, 0)
	for rows.Next() {
		p, err := scanPitch(rows)
		if err != nil {
			return nil, err
		}
		pitches = append(pitches, p)
	}
	return pitches, rows.Err()
}

func (s *PostgresStore) Update(ctx context.Context, userID, id string, u PitchUpdate) (Pitch, error) {
	const query = `
		update public.pitches set
			name = coalesce($3, name),
			latitude = coalesce($4, latitude),
			longitude = coalesce($5, longitude),
			type = coalesce($6, type),
			surface = coalesce($7, surface),
			length_m = coalesce($8, length_m),
			width_m = coalesce($9, width_m),
			measurement_method = coalesce($10, measurement_method),
			indoor = coalesce($11, indoor),
			notes = coalesce($12, notes),
			updated_at = now()
		where id = $1 and user_id = $2
		returning ` + pitchColumns

	p, err := scanPitch(s.pool.QueryRow(ctx, query,
		id, userID,
		u.Name, u.Latitude, u.Longitude, u.Type, u.Surface,
		u.LengthM, u.WidthM, u.MeasurementMethod, u.Indoor, u.Notes,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Pitch{}, ErrNotFound
	}
	return p, err
}

func (s *PostgresStore) Delete(ctx context.Context, userID, id string) error {
	tag, err := s.pool.Exec(ctx, `delete from public.pitches where id = $1 and user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) OwnedByUser(ctx context.Context, userID, id string) (bool, error) {
	const query = `select exists(select 1 from public.pitches where id = $1 and user_id = $2)`
	var exists bool
	err := s.pool.QueryRow(ctx, query, id, userID).Scan(&exists)
	return exists, err
}

type scanRow interface {
	Scan(dest ...any) error
}

func scanPitch(r scanRow) (Pitch, error) {
	var p Pitch
	err := r.Scan(
		&p.ID, &p.UserID, &p.Name,
		&p.Latitude, &p.Longitude, &p.Type, &p.Surface,
		&p.LengthM, &p.WidthM, &p.MeasurementMethod, &p.Indoor, &p.Notes,
		&p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}
