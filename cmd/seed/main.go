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

// Pitches seeded for the demo user; sessions reference these by index.
var demoPitchIDs = [3]string{
	"b2000002-0000-4000-8000-000000000001",
	"b2000002-0000-4000-8000-000000000002",
	"b2000002-0000-4000-8000-000000000003",
}

type demoPitch struct {
	id                   string
	name                 string
	pType                string
	surface              string
	lengthM              float64
	widthM               float64
	measurementMethod    string
	latOffset, lonOffset float64
}

type demoMatch struct {
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
	surface     string
	position    string
	matchTag    string
	result      string
	feeling     int
	opponent    string
	pitchIdx    int
	structured  bool
	rating      float64
}

// demoSessionID derives a deterministic UUID for the i-th seeded session.
func demoSessionID(i int) string {
	return fmt.Sprintf("a1000001-0000-4000-8000-%012d", i+1)
}

// positionXBias biases the heatmap toward the area a position usually occupies.
func positionXBias(position string) float64 {
	switch position {
	case "Goalkeeper":
		return -0.34
	case "Defender":
		return -0.22
	case "Full-back":
		return -0.12
	case "Midfielder":
		return 0
	case "Winger":
		return 0.12
	case "Forward":
		return 0.22
	default:
		return 0
	}
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

	log.Printf("done — profile + %d pitches + %d sessions (context, GPS heatmap, structured halves) for %s / %s", len(demoPitchIDs), sessionCount, demoEmail, demoPassword)
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

func seedPitches(ctx context.Context, tx pgx.Tx, userID string) error {
	pitches := []demoPitch{
		{demoPitchIDs[0], "Estadio Central", "11-a-side", "Natural grass", 105, 68, "walk", 0, 0},
		{demoPitchIDs[1], "Canchas El Norte", "7-a-side", "Artificial turf", 55, 35, "manual", 0.0040, 0.0040},
		{demoPitchIDs[2], "Polideportivo Sur", "5-a-side", "Indoor", 40, 20, "manual", -0.0030, 0.0020},
	}
	const q = `
		insert into public.pitches (id, user_id, name, latitude, longitude, type, surface, length_m, width_m, measurement_method,
			indoor, notes)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		on conflict (id) do update set
			name = excluded.name, latitude = excluded.latitude, longitude = excluded.longitude,
			type = excluded.type, surface = excluded.surface, length_m = excluded.length_m,
			width_m = excluded.width_m, measurement_method = excluded.measurement_method,
			indoor = excluded.indoor, notes = excluded.notes, updated_at = now()`
	for _, p := range pitches {
		indoor := p.surface == "Indoor"
		notes := "Demo court — synthetic data for previews."
		if _, err := tx.Exec(ctx, q, p.id, userID, p.name,
			demoBaseLat+p.latOffset, demoBaseLon+p.lonOffset, p.pType, p.surface, p.lengthM, p.widthM, p.measurementMethod,
			indoor, notes,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedSessions(ctx context.Context, pool *pgxpool.Pool, userID string) (int, error) {
	// ~16 weeks of varied matches: modes, surfaces, positions (≥3 in 4 positions
	// for position insights), match types/tags, full context + opponent, rating
	// trending up recently, several structured halves, and many sessions sharing
	// pitch 0 (same-court comparison). Dates have no weekly gap → a live streak.
	matches := []demoMatch{
		{1, 92, 10200, 156, 182, 28.5, 16, 86, 790, "structured", "11-a-side", "Natural grass", "Midfielder", "league", "3-1 W", 5, "Los Tigres", 0, true, 90.0},
		{4, 60, 7600, 150, 176, 25.0, 11, 78, 600, "quick", "7-a-side", "Artificial turf", "Forward", "friendly", "2-2 D", 4, "FC Norte", 1, false, 86.0},
		{7, 90, 9400, 152, 178, 26.3, 12, 81, 740, "structured", "11-a-side", "Natural grass", "Midfielder", "league", "1-0 W", 5, "Real Sur", 0, true, 88.0},
		{10, 45, 5200, 144, 170, 23.1, 7, 70, 460, "training", "5-a-side", "Indoor", "Winger", "training", "", 3, "", 2, false, 82.0},
		{13, 88, 8900, 149, 175, 25.6, 10, 79, 700, "structured", "11-a-side", "Natural grass", "Defender", "league", "0-1 L", 2, "Atlético Centro", 0, true, 80.0},
		{17, 55, 7100, 147, 172, 24.4, 9, 75, 560, "quick", "7-a-side", "Artificial turf", "Midfielder", "friendly", "4-3 W", 5, "Halcones", 1, false, 84.0},
		{20, 90, 9100, 151, 177, 25.9, 11, 80, 720, "structured", "9-a-side", "Natural grass", "Forward", "league", "2-1 W", 4, "Deportivo Olancho", 0, true, 83.0},
		{24, 50, 5600, 142, 168, 22.8, 6, 68, 470, "training", "5-a-side", "Indoor", "Winger", "training", "", 4, "", 2, false, 76.0},
		{27, 60, 7400, 146, 171, 24.0, 9, 74, 580, "quick", "7-a-side", "Concrete", "Midfielder", "friendly", "1-1 D", 3, "Barrio Unido", 1, false, 75.0},
		{31, 90, 8700, 148, 174, 25.2, 10, 78, 690, "structured", "11-a-side", "Natural grass", "Defender", "league", "3-0 W", 5, "San Pedro FC", 0, true, 79.0},
		{34, 45, 5000, 140, 166, 22.0, 5, 66, 440, "training", "5-a-side", "Indoor", "Midfielder", "training", "", 3, "", 2, false, 70.0},
		{38, 58, 7000, 145, 170, 23.7, 8, 73, 560, "quick", "7-a-side", "Artificial turf", "Forward", "friendly", "0-2 L", 2, "Los Lobos", 1, false, 72.0},
		{41, 90, 8500, 147, 173, 24.8, 9, 77, 680, "structured", "11-a-side", "Natural grass", "Midfielder", "league", "2-2 D", 3, "Club Atlético", 0, true, 77.0},
		{45, 52, 5400, 141, 167, 22.5, 6, 67, 450, "training", "5-a-side", "Concrete", "Defender", "training", "", 4, "", 2, false, 68.0},
		{48, 56, 6800, 144, 169, 23.4, 8, 72, 540, "quick", "9-a-side", "Artificial turf", "Forward", "friendly", "3-3 D", 4, "Tegus FC", 1, false, 74.0},
		{52, 90, 8300, 146, 172, 24.5, 9, 76, 670, "structured", "11-a-side", "Natural grass", "Midfielder", "league", "1-2 L", 2, "Real Sur", 0, true, 73.0},
		{56, 48, 5100, 139, 165, 21.8, 5, 65, 430, "training", "5-a-side", "Indoor", "Winger", "training", "", 3, "", 2, false, 66.0},
		{59, 60, 7200, 145, 170, 23.9, 9, 73, 570, "quick", "7-a-side", "Concrete", "Midfielder", "friendly", "2-0 W", 5, "Halcones", 1, false, 71.0},
		{63, 88, 8000, 145, 171, 24.0, 8, 75, 650, "structured", "11-a-side", "Natural grass", "Forward", "league", "0-0 D", 3, "Atlético Centro", 0, true, 69.0},
		{67, 50, 5300, 140, 166, 22.2, 6, 66, 450, "training", "5-a-side", "Indoor", "Defender", "training", "", 3, "", 2, false, 64.0},
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	ids := make([]string, len(matches))
	for i := range matches {
		ids[i] = demoSessionID(i)
	}
	if _, err := tx.Exec(ctx, `delete from public.session_path where session_id = any($1)`, ids); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `delete from public.session_samples where session_id = any($1)`, ids); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `delete from public.sessions where id = any($1)`, ids); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `delete from public.pitches where id = any($1)`, demoPitchIDs[:]); err != nil {
		return 0, err
	}
	if err := seedPitches(ctx, tx, userID); err != nil {
		return 0, err
	}

	const insertSession = `
		insert into public.sessions (
			id, user_id, started_at, ended_at, duration_s, distance_m,
			hr_avg, hr_max, speed_max_kmh, sprints, intensity, calories_kcal, source,
			mode, match_type, halftime_offset_s, match_rating, position, surface, match_tag,
			feeling, result, opponent, pitch_id,
			outcome, score, competition, goals, assists
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29)`

	now := time.Now().UTC()
	for i, m := range matches {
		id := demoSessionID(i)
		start := now.AddDate(0, 0, -m.daysAgo).Truncate(time.Minute)
		duration := time.Duration(m.durationMin) * time.Minute
		end := start.Add(duration)
		durationS := int(duration.Seconds())

		var halftimeS *int
		if m.structured {
			hs := durationS / 2
			halftimeS = &hs
		}

		// Derive the structured result from the legacy "3-1 W" string.
		outcome, score := parseResult(m.result)
		var goals, assists *int
		competition := deriveCompetition(m.matchTag, i)
		if outcome != "" {
			g := i % 3
			a := (i + 1) % 2
			goals = &g
			assists = &a
		}

		if _, err := tx.Exec(ctx, insertSession,
			id, userID, start, end, durationS, m.distanceM,
			m.hrAvg, m.hrMax, m.speedMax, m.sprints, m.intensity, m.calories, "watch",
			m.mode, m.matchType, halftimeS, ptrFloat(m.rating), m.position, m.surface, m.matchTag,
			m.feeling, nilIfEmpty(m.result), nilIfEmpty(m.opponent), demoPitchIDs[m.pitchIdx],
			nilIfEmpty(outcome), nilIfEmpty(score), nilIfEmpty(competition), goals, assists,
		); err != nil {
			return 0, err
		}

		if err := insertSamples(ctx, tx, id, m.hrAvg, m.hrMax, durationS, halftimeS); err != nil {
			return 0, err
		}
		if err := insertPath(ctx, tx, id, i, durationS, m.position); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(matches), nil
}

func insertSamples(ctx context.Context, tx pgx.Tx, sessionID string, hrAvg, hrMax, durationS int, halftimeS *int) error {
	rows := make([][]any, 0, durationS/10+1)
	for offset := 0; offset <= durationS; offset += 10 {
		hr := demoHR(offset, hrAvg)
		if hr > hrMax {
			hr = hrMax
		}
		speed := demoSpeedKmh(offset, durationS)
		var half *int
		if halftimeS != nil {
			h := 1
			if offset >= *halftimeS {
				h = 2
			}
			half = &h
		}
		rows = append(rows, []any{sessionID, offset, hr, speed, half})
	}
	_, err := tx.CopyFrom(ctx,
		pgx.Identifier{"public", "session_samples"},
		[]string{"session_id", "t_offset_s", "hr", "speed_kmh", "half"},
		pgx.CopyFromRows(rows),
	)
	return err
}

func insertPath(ctx context.Context, tx pgx.Tx, sessionID string, sessionIndex, durationS int, position string) error {
	xHome := clamp01(0.5 + positionXBias(position))
	yHome := positionYHome(position, sessionIndex)
	points := demoPathPoints(durationS, sessionIndex, xHome, yHome)

	rows := make([][]any, 0, len(points))
	for i, p := range points {
		lat, lon := pitchToGPS(p[0], p[1])
		rows = append(rows, []any{sessionID, i * 5, lat, lon})
	}
	_, err := tx.CopyFrom(ctx,
		pgx.Identifier{"public", "session_path"},
		[]string{"session_id", "t_offset_s", "latitude", "longitude"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// positionYHome returns the player's home width (0…1). Wide roles sit near a
// touchline (alternating side per session); central roles stay middle.
func positionYHome(position string, sessionIndex int) float64 {
	wide := func(a, b float64) float64 {
		if sessionIndex%2 == 0 {
			return a
		}
		return b
	}
	switch position {
	case "Winger":
		return wide(0.24, 0.76)
	case "Full-back":
		return wide(0.20, 0.80)
	default:
		return 0.5
	}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// parseResult turns a legacy "3-1 W" string into (outcome, score). Empty input
// yields empty strings.
func parseResult(result string) (outcome, score string) {
	parts := strings.Fields(result)
	if len(parts) < 2 {
		return "", ""
	}
	score = parts[0]
	switch parts[1] {
	case "W":
		outcome = "win"
	case "D":
		outcome = "draw"
	case "L":
		outcome = "loss"
	}
	return outcome, score
}

// deriveCompetition maps the legacy match_tag to a competition, promoting a
// few league matches to "tournament" so the seed exercises all enum values.
func deriveCompetition(matchTag string, index int) string {
	switch matchTag {
	case "friendly":
		return "friendly"
	case "league":
		if index%4 == 0 {
			return "tournament"
		}
		return "league"
	case "training":
		return "training"
	default:
		return ""
	}
}

func ptrFloat(v float64) *float64 {
	return &v
}
