package goal

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

const goalColumns = `id, user_id, metric, period, target, created_at, achieved_at, archived`

func scanGoal(r scanRow) (Goal, error) {
	var g Goal
	err := r.Scan(&g.ID, &g.UserID, &g.Metric, &g.Period, &g.Target, &g.CreatedAt, &g.AchievedAt, &g.Archived)
	return g, err
}

type scanRow interface {
	Scan(dest ...any) error
}

func (s *PostgresStore) Create(ctx context.Context, userID string, n NewGoal) (Goal, error) {
	const query = `
		insert into public.personal_goals (user_id, metric, period, target)
		values ($1, $2, $3, $4)
		returning ` + goalColumns
	return scanGoal(s.pool.QueryRow(ctx, query, userID, n.Metric, n.Period, n.Target))
}

func (s *PostgresStore) List(ctx context.Context, userID string) ([]Goal, error) {
	const query = `
		select ` + goalColumns + `
		from public.personal_goals
		where user_id = $1 and archived = false
		order by created_at desc`
	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var goals []Goal
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

func (s *PostgresStore) Get(ctx context.Context, userID, id string) (Goal, error) {
	const query = `
		select ` + goalColumns + `
		from public.personal_goals
		where id = $1 and user_id = $2`
	g, err := scanGoal(s.pool.QueryRow(ctx, query, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Goal{}, ErrNotFound
	}
	return g, err
}

func (s *PostgresStore) Update(ctx context.Context, userID, id string, u GoalUpdate) (Goal, error) {
	const query = `
		update public.personal_goals set
			metric    = coalesce($3, metric),
			period    = coalesce($4, period),
			target    = coalesce($5, target),
			archived  = coalesce($6, archived)
		where id = $1 and user_id = $2
		returning ` + goalColumns
	g, err := scanGoal(s.pool.QueryRow(ctx, query, id, userID, u.Metric, u.Period, u.Target, u.Archived))
	if errors.Is(err, pgx.ErrNoRows) {
		return Goal{}, ErrNotFound
	}
	return g, err
}

func (s *PostgresStore) Delete(ctx context.Context, userID, id string) error {
	const query = `delete from public.personal_goals where id = $1 and user_id = $2`
	tag, err := s.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) Achieve(ctx context.Context, userID, id string, at time.Time) error {
	const query = `
		update public.personal_goals
		set achieved_at = $3
		where id = $1 and user_id = $2 and achieved_at is null`
	tag, err := s.pool.Exec(ctx, query, id, userID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
