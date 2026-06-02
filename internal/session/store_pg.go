package session

import (
	"context"
	"errors"
	"time"

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
	hr_avg, hr_max, speed_max_kmh, sprints, intensity, calories_kcal, source,
	mode, halftime_offset_s, match_type, surface, position, result, feeling,
	match_tag, pitch_id, match_rating, created_at`

func (s *PostgresStore) Create(ctx context.Context, userID string, n New) (Session, error) {
	const query = `
		insert into public.sessions
			(user_id, started_at, ended_at, duration_s, distance_m,
			 hr_avg, hr_max, speed_max_kmh, sprints, intensity, calories_kcal, source,
			 mode, halftime_offset_s, match_rating, pitch_id)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		returning ` + sessionColumns

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)

	sess, err := scanSession(tx.QueryRow(ctx, query,
		userID, n.StartedAt, n.EndedAt, n.DurationS, n.DistanceM,
		n.HRAvg, n.HRMax, n.SpeedMaxKMH, n.Sprints, n.Intensity, n.CaloriesKcal, n.Source,
		n.Mode, n.HalftimeOffsetS, n.MatchRating, n.PitchID,
	))
	if err != nil {
		return Session{}, err
	}

	if len(n.Samples) > 0 {
		rows := make([][]any, len(n.Samples))
		for i, smp := range n.Samples {
			rows[i] = []any{sess.ID, smp.TOffsetS, smp.HR, smp.SpeedKMH, smp.Half}
		}
		_, err = tx.CopyFrom(ctx,
			pgx.Identifier{"public", "session_samples"},
			[]string{"session_id", "t_offset_s", "hr", "speed_kmh", "half"},
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
	path, err := s.loadPath(ctx, sess.ID)
	if err != nil {
		return Session{}, err
	}
	sess.Path = path
	return sess, nil
}

func (s *PostgresStore) Update(ctx context.Context, userID, id string, u Update) (Session, error) {
	const query = `
		update public.sessions set
			started_at = $3, ended_at = $4, duration_s = $5, distance_m = $6,
			hr_avg = $7, hr_max = $8, speed_max_kmh = $9, sprints = $10,
			intensity = $11, calories_kcal = $12
		where id = $1 and user_id = $2
		returning ` + sessionColumns

	sess, err := scanSession(s.pool.QueryRow(ctx, query,
		id, userID, u.StartedAt, u.EndedAt, u.DurationS, u.DistanceM,
		u.HRAvg, u.HRMax, u.SpeedMaxKMH, u.Sprints, u.Intensity, u.CaloriesKcal,
	))
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
	path, err := s.loadPath(ctx, sess.ID)
	if err != nil {
		return Session{}, err
	}
	sess.Path = path
	return sess, nil
}

func (s *PostgresStore) UpdateContext(ctx context.Context, userID, id string, cu ContextUpdate) (Session, error) {
	const query = `
		update public.sessions set
			match_type = $3, surface = $4, position = $5, result = $6,
			feeling = $7, match_tag = $8, pitch_id = $9
		where id = $1 and user_id = $2
		returning ` + sessionColumns

	sess, err := scanSession(s.pool.QueryRow(ctx, query,
		id, userID, cu.MatchType, cu.Surface, cu.Position, cu.Result,
		cu.Feeling, cu.MatchTag, cu.PitchID,
	))
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
	path, err := s.loadPath(ctx, sess.ID)
	if err != nil {
		return Session{}, err
	}
	sess.Path = path
	return sess, nil
}

func (s *PostgresStore) Delete(ctx context.Context, userID, id string) error {
	const query = `delete from public.sessions where id = $1 and user_id = $2`
	tag, err := s.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) GetPersonalRecords(ctx context.Context, userID string) (PersonalRecords, error) {
	const query = `
		with ranked as (
			select
				distance_m, duration_s::float8, speed_max_kmh, sprints::float8,
				intensity, match_rating::float8, hr_max::float8, calories_kcal,
				id, started_at
			from public.sessions
			where user_id = $1
		)
		select
			(select distance_m   from ranked where distance_m   is not null order by distance_m   desc nulls last limit 1) as best_distance,
			(select id           from ranked where distance_m   is not null order by distance_m   desc nulls last limit 1) as distance_sid,
			(select started_at   from ranked where distance_m   is not null order by distance_m   desc nulls last limit 1) as distance_date,
			(select duration_s   from ranked where duration_s   is not null order by duration_s   desc nulls last limit 1) as best_duration,
			(select id           from ranked where duration_s   is not null order by duration_s   desc nulls last limit 1) as duration_sid,
			(select started_at   from ranked where duration_s   is not null order by duration_s   desc nulls last limit 1) as duration_date,
			(select speed_max_kmh from ranked where speed_max_kmh is not null order by speed_max_kmh desc nulls last limit 1) as best_speed,
			(select id           from ranked where speed_max_kmh is not null order by speed_max_kmh desc nulls last limit 1) as speed_sid,
			(select started_at   from ranked where speed_max_kmh is not null order by speed_max_kmh desc nulls last limit 1) as speed_date,
			(select sprints      from ranked where sprints       is not null order by sprints      desc nulls last limit 1) as best_sprints,
			(select id           from ranked where sprints       is not null order by sprints      desc nulls last limit 1) as sprints_sid,
			(select started_at   from ranked where sprints       is not null order by sprints      desc nulls last limit 1) as sprints_date,
			(select intensity    from ranked where intensity     is not null order by intensity    desc nulls last limit 1) as best_intensity,
			(select id           from ranked where intensity     is not null order by intensity    desc nulls last limit 1) as intensity_sid,
			(select started_at   from ranked where intensity     is not null order by intensity    desc nulls last limit 1) as intensity_date,
			(select match_rating from ranked where match_rating  is not null order by match_rating desc nulls last limit 1) as best_rating,
			(select id           from ranked where match_rating  is not null order by match_rating desc nulls last limit 1) as rating_sid,
			(select started_at   from ranked where match_rating  is not null order by match_rating desc nulls last limit 1) as rating_date,
			(select hr_max       from ranked where hr_max        is not null order by hr_max       desc nulls last limit 1) as best_hr_max,
			(select id           from ranked where hr_max        is not null order by hr_max       desc nulls last limit 1) as hr_max_sid,
			(select started_at   from ranked where hr_max        is not null order by hr_max       desc nulls last limit 1) as hr_max_date,
			(select calories_kcal from ranked where calories_kcal is not null order by calories_kcal desc nulls last limit 1) as best_calories,
			(select id           from ranked where calories_kcal is not null order by calories_kcal desc nulls last limit 1) as calories_sid,
			(select started_at   from ranked where calories_kcal is not null order by calories_kcal desc nulls last limit 1) as calories_date
	`

	var (
		bestDistance   float64
		distanceSid    *string
		distanceDate   *time.Time
		bestDuration   float64
		durationSid    *string
		durationDate   *time.Time
		bestSpeed      *float64
		speedSid       *string
		speedDate      *time.Time
		bestSprints    float64
		sprintsSid     *string
		sprintsDate    *time.Time
		bestIntensity  *float64
		intensitySid   *string
		intensityDate  *time.Time
		bestRating     *float64
		ratingSid      *string
		ratingDate     *time.Time
		bestHRMax      *float64
		hrMaxSid       *string
		hrMaxDate      *time.Time
		bestCalories   *float64
		caloriesSid    *string
		caloriesDate   *time.Time
	)

	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&bestDistance, &distanceSid, &distanceDate,
		&bestDuration, &durationSid, &durationDate,
		&bestSpeed, &speedSid, &speedDate,
		&bestSprints, &sprintsSid, &sprintsDate,
		&bestIntensity, &intensitySid, &intensityDate,
		&bestRating, &ratingSid, &ratingDate,
		&bestHRMax, &hrMaxSid, &hrMaxDate,
		&bestCalories, &caloriesSid, &caloriesDate,
	)
	if err != nil {
		return PersonalRecords{}, err
	}

	records := make([]RecordEntry, 0, 9)

	add := func(metric string, value float64, sid *string, date *time.Time) {
		if sid != nil && date != nil {
			records = append(records, RecordEntry{
				Metric:    metric,
				Value:     value,
				SessionID: *sid,
				StartedAt: *date,
			})
		}
	}

	add("distance_m", bestDistance, distanceSid, distanceDate)
	add("duration_s", bestDuration, durationSid, durationDate)
	if bestSpeed != nil {
		add("speed_max_kmh", *bestSpeed, speedSid, speedDate)
	}
	add("sprints", bestSprints, sprintsSid, sprintsDate)
	if bestIntensity != nil {
		add("intensity", *bestIntensity, intensitySid, intensityDate)
	}
	if bestRating != nil {
		add("match_rating", *bestRating, ratingSid, ratingDate)
	}
	if bestHRMax != nil {
		add("hr_max", *bestHRMax, hrMaxSid, hrMaxDate)
	}
	if bestCalories != nil {
		add("calories_kcal", *bestCalories, caloriesSid, caloriesDate)
	}

	return PersonalRecords{Records: records}, nil
}

func (s *PostgresStore) loadSamples(ctx context.Context, sessionID string) ([]Sample, error) {
	const query = `
		select t_offset_s, hr, speed_kmh, half
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
		if err := rows.Scan(&smp.TOffsetS, &smp.HR, &smp.SpeedKMH, &smp.Half); err != nil {
			return nil, err
		}
		samples = append(samples, smp)
	}
	return samples, rows.Err()
}

func (s *PostgresStore) loadPath(ctx context.Context, sessionID string) ([]PathPoint, error) {
	const query = `
		select t_offset_s, latitude, longitude
		from public.session_path
		where session_id = $1
		order by t_offset_s`

	rows, err := s.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	path := make([]PathPoint, 0)
	for rows.Next() {
		var pt PathPoint
		if err := rows.Scan(&pt.TOffsetS, &pt.Latitude, &pt.Longitude); err != nil {
			return nil, err
		}
		path = append(path, pt)
	}
	return path, rows.Err()
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
		&s.Source, &s.Mode, &s.HalftimeOffsetS,
		&s.MatchType, &s.Surface, &s.Position, &s.Result, &s.Feeling,
		&s.MatchTag, &s.PitchID, &s.MatchRating, &s.CreatedAt,
	)
	return s, err
}
