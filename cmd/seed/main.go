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
	id         string
	daysAgo    int
	durationMin int
	distanceM  float64
	hrAvg      int
	hrMax      int
	speedMax   float64
	sprints    int
	intensity  float64
	calories   float64
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

	log.Printf("done — profile + %d sessions for %s / %s", sessionCount, demoEmail, demoPassword)
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
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("create user: %s %s", res.Status, string(raw))
	}

	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("create user: missing id in response")
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
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("sync user: %s %s", res.Status, string(raw))
	}
	return nil
}

func userIDFromDB(ctx context.Context, pool *pgxpool.Pool, email string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `select id::text from auth.users where email = $1`, email).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("lookup auth.users for %s: %w", email, err)
	}
	return id, nil
}

func findUserByEmail(ctx context.Context, baseURL, serviceKey, email string) (string, error) {
	endpoint, err := url.Parse(baseURL + "/auth/v1/admin/users")
	if err != nil {
		return "", err
	}
	q := endpoint.Query()
	q.Set("email", email)
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
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
	if res.StatusCode == http.StatusNotFound {
		return "", nil
	}
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("list users: %s %s", res.Status, string(raw))
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

func seedProfile(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	const q = `
		insert into public.profiles (id, display_name, preferred_position, height_cm, weight_kg)
		values ($1, 'Geovany', 'Midfielder', 170, 70)
		on conflict (id) do update set
			display_name = excluded.display_name,
			preferred_position = excluded.preferred_position,
			height_cm = excluded.height_cm,
			weight_kg = excluded.weight_kg,
			updated_at = now()`
	_, err := pool.Exec(ctx, q, userID)
	return err
}

func seedSessions(ctx context.Context, pool *pgxpool.Pool, userID string) (int, error) {
	matches := []demoMatch{
		{demoSessionIDs[0], 35, 82, 8120, 138, 162, 24.2, 7, 68, 620},
		{demoSessionIDs[1], 28, 90, 9050, 145, 171, 25.1, 11, 74, 710},
		{demoSessionIDs[2], 21, 85, 7680, 141, 165, 23.4, 6, 70, 590},
		{demoSessionIDs[3], 14, 88, 8340, 148, 175, 25.8, 10, 76, 655},
		{demoSessionIDs[4], 7, 91, 9420, 152, 178, 26.3, 12, 81, 740},
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `delete from public.session_samples where session_id = any($1)`, demoSessionIDs[:]); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `delete from public.sessions where id = any($1)`, demoSessionIDs[:]); err != nil {
		return 0, err
	}

	const insertSession = `
		insert into public.sessions (
			id, user_id, started_at, ended_at, duration_s, distance_m,
			hr_avg, hr_max, speed_max_kmh, sprints, intensity, calories_kcal, source
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	const insertSample = `
		insert into public.session_samples (session_id, t_offset_s, hr, speed_kmh)
		values ($1, $2, $3, $4)`

	now := time.Now().UTC()
	for _, m := range matches {
		start := now.AddDate(0, 0, -m.daysAgo).Truncate(time.Minute)
		duration := time.Duration(m.durationMin) * time.Minute
		end := start.Add(duration)
		durationS := int(duration.Seconds())

		if _, err := tx.Exec(ctx, insertSession,
			m.id, userID, start, end, durationS, m.distanceM,
			m.hrAvg, m.hrMax, m.speedMax, m.sprints, m.intensity, m.calories, "watch",
		); err != nil {
			return 0, err
		}

		for offset := 0; offset <= durationS; offset += 300 {
			hr := m.hrAvg - 15 + (offset/300)*3 + (offset % 120 / 20)
			speed := 8.0 + (float64(offset)/float64(durationS))*12.0
			if _, err := tx.Exec(ctx, insertSample, m.id, offset, hr, speed); err != nil {
				return 0, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(matches), nil
}
