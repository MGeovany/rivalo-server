// Seed demo user marlongeo1999+mid@gmail.com with profile and 5 fake watch sessions.
//
// Usage (from repo root, with .env configured):
//
//	make seed
//
// Requires DATABASE_URL, SUPABASE_URL, and SUPABASE_SERVICE_ROLE_KEY in .env.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

const (
	demoEmail    = "marlongeo1999+mid@gmail.com"
	demoPassword = "Rivalo@123"
)

var demoSessionIDs = [5]string{
	"a1000001-0000-4000-8000-000000000001",
	"a1000001-0000-4000-8000-000000000002",
	"a1000001-0000-4000-8000-000000000003",
	"a1000001-0000-4000-8000-000000000004",
	"a1000001-0000-4000-8000-000000000005",
}

type demoMatch struct {
	id          string
	index       int
	daysAgo     int
	durationMin int
	distanceM   float64
	hrAvg       int
	hrMax       int
	speedMax    float64
	sprints     int
	intensity   float64
	calories    float64
	mode        string
	matchType   string
	halftimeS   *int
	matchRating *float64
}

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	supabaseURL := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	serviceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if dbURL == "" || supabaseURL == "" || serviceKey == "" {
		log.Fatal("DATABASE_URL, SUPABASE_URL, and SUPABASE_SERVICE_ROLE_KEY must be set in .env")
	}

	if _, err := ensureAuthUser(ctx, supabaseURL, serviceKey, demoEmail, demoPassword); err != nil {
		log.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	userID, err := userIDFromDB(ctx, pool, demoEmail)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("auth user %s id=%s", demoEmail, userID)

	if err := seedProfile(ctx, pool, userID); err != nil {
		log.Fatal(err)
	}
	sessionCount, err := seedSessions(ctx, pool, userID)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("done — profile + %d sessions (dense samples + GPS path) for %s / %s", sessionCount, demoEmail, demoPassword)
}

func ensureAuthUser(ctx context.Context, baseURL, serviceKey, email, password string) (string, error) {
	existing, err := findUserByEmail(ctx, baseURL, serviceKey, email)
	if err != nil {
		return "", err
	}
	if existing != "" {
		log.Printf("auth user already exists — syncing password and email_confirm")
		if err := syncAuthUser(ctx, baseURL, serviceKey, existing, password); err != nil {
			return "", err
		}
		return existing, nil
	}

	body, _ := json.Marshal(map[string]any{
		"email":         email,
		"password":      password,
		"email_confirm": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/auth/v1/admin/users", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+serviceKey)
	req.Header.Set("apikey", serviceKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("create user: %s %s", res.Status, raw)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func syncAuthUser(ctx context.Context, baseURL, serviceKey, userID, password string) error {
	body, _ := json.Marshal(map[string]any{
		"password":      password,
		"email_confirm": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, baseURL+"/auth/v1/admin/users/"+userID, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+serviceKey)
	req.Header.Set("apikey", serviceKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		raw, _ := io.ReadAll(res.Body)
		return fmt.Errorf("sync user: %s %s", res.Status, raw)
	}
	return nil
}

func findUserByEmail(ctx context.Context, baseURL, serviceKey, email string) (string, error) {
	u := baseURL + "/auth/v1/admin/users?email=" + url.QueryEscape(email)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+serviceKey)
	req.Header.Set("apikey", serviceKey)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("list users: %s %s", res.Status, raw)
	}

	var out struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Users) == 0 {
		return "", nil
	}
	return out.Users[0].ID, nil
}

func userIDFromDB(ctx context.Context, pool *pgxpool.Pool, email string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `select id from auth.users where email = $1`, email).Scan(&id)
	return id, err
}

func seedProfile(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	const q = `
		insert into public.profiles (id, display_name, preferred_position, height_cm, weight_kg, birth_year)
		values ($1, 'Geovany', 'Midfielder', 170, 70, 1999)
		on conflict (id) do update set
			display_name = excluded.display_name,
			preferred_position = excluded.preferred_position,
			height_cm = excluded.height_cm,
			weight_kg = excluded.weight_kg,
			birth_year = excluded.birth_year,
			updated_at = now()`
	_, err := pool.Exec(ctx, q, userID)
	return err
}

func seedSessions(ctx context.Context, pool *pgxpool.Pool, userID string) (int, error) {
	halftime := 2730
	rating := 78.5
	matches := []demoMatch{
		{demoSessionIDs[0], 0, 35, 82, 8120, 138, 162, 24.2, 7, 68, 620, "quick", "7-a-side", nil, nil},
		{demoSessionIDs[1], 1, 28, 90, 9050, 145, 171, 25.1, 11, 74, 710, "quick", "7-a-side", nil, nil},
		{demoSessionIDs[2], 2, 21, 85, 7680, 141, 165, 23.4, 6, 70, 590, "quick", "9-a-side", nil, nil},
		{demoSessionIDs[3], 3, 14, 88, 8340, 148, 175, 25.8, 10, 76, 655, "structured", "11-a-side", &halftime, ptrFloat(72.0)},
		{demoSessionIDs[4], 4, 7, 91, 9420, 152, 178, 26.3, 12, 81, 740, "structured", "11-a-side", &halftime, &rating},
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `delete from public.session_path where session_id = any($1)`, demoSessionIDs[:]); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `delete from public.session_samples where session_id = any($1)`, demoSessionIDs[:]); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `delete from public.sessions where id = any($1)`, demoSessionIDs[:]); err != nil {
		return 0, err
	}

	const insertSession = `
		insert into public.sessions (
			id, user_id, started_at, ended_at, duration_s, distance_m,
			hr_avg, hr_max, speed_max_kmh, sprints, intensity, calories_kcal, source,
			mode, match_type, halftime_offset_s, match_rating, position, surface, match_tag, feeling
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`

	now := time.Now().UTC()
	for _, m := range matches {
		start := now.AddDate(0, 0, -m.daysAgo).Truncate(time.Minute)
		duration := time.Duration(m.durationMin) * time.Minute
		end := start.Add(duration)
		durationS := int(duration.Seconds())

		if _, err := tx.Exec(ctx, insertSession,
			m.id, userID, start, end, durationS, m.distanceM,
			m.hrAvg, m.hrMax, m.speedMax, m.sprints, m.intensity, m.calories, "watch",
			m.mode, m.matchType, m.halftimeS, m.matchRating,
			"Midfielder", "Artificial turf", "league", 4,
		); err != nil {
			return 0, err
		}

		if err := insertSamples(ctx, tx, m, durationS); err != nil {
			return 0, err
		}
		if err := insertPath(ctx, tx, m.id, m.index, durationS); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(matches), nil
}

func insertSamples(ctx context.Context, tx pgx.Tx, m demoMatch, durationS int) error {
	rows := make([][]any, 0, durationS/10+1)
	for offset := 0; offset <= durationS; offset += 10 {
		hr := demoHR(offset, m.hrAvg)
		if hr > m.hrMax {
			hr = m.hrMax
		}
		speed := demoSpeedKmh(offset, durationS)
		var half *int
		if m.halftimeS != nil {
			h := 1
			if offset >= *m.halftimeS {
				h = 2
			}
			half = &h
		}
		rows = append(rows, []any{m.id, offset, hr, speed, half})
	}
	_, err := tx.CopyFrom(ctx,
		pgx.Identifier{"public", "session_samples"},
		[]string{"session_id", "t_offset_s", "hr", "speed_kmh", "half"},
		pgx.CopyFromRows(rows),
	)
	return err
}

func insertPath(ctx context.Context, tx pgx.Tx, sessionID string, sessionIndex, durationS int) error {
	rows := make([][]any, 0, durationS/5+1)
	for offset := 0; offset <= durationS; offset += 5 {
		x, y := demoPitchXY(offset, durationS, sessionIndex)
		lat, lon := pitchToGPS(x, y)
		rows = append(rows, []any{sessionID, offset, lat, lon})
	}
	_, err := tx.CopyFrom(ctx,
		pgx.Identifier{"public", "session_path"},
		[]string{"session_id", "t_offset_s", "latitude", "longitude"},
		pgx.CopyFromRows(rows),
	)
	return err
}

func ptrFloat(v float64) *float64 {
	return &v
}
