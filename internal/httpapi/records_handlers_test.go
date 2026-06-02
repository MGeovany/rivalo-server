package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/MGeovany/rivalo-server/internal/session"
)

func TestGetRecords_NoSessions_200(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-empty", time.Now().Add(time.Hour))
	rec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/records", "Bearer "+token, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var pr session.PersonalRecords
	if err := json.NewDecoder(rec.Body).Decode(&pr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(pr.Records) != 0 {
		t.Fatalf("got %d records, want 0", len(pr.Records))
	}
}

func TestGetRecords_WithSessions_200(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-records", time.Now().Add(time.Hour))

	start := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)

	// Session 1: lower distance, lower sprints, no HR/speed
	s1 := validSessionBody()
	s1.StartedAt = start
	s1.EndedAt = start.Add(60 * time.Minute)
	s1.DurationS = 3600
	s1.DistanceM = 5000
	s1.Sprints = 10
	s1.HRAvg = intPtr(130)
	s1.HRMax = intPtr(160)
	speed1 := 22.0
	s1.SpeedMaxKMH = &speed1
	s1.Intensity = floatPtr(55.0)
	s1.CaloriesKcal = floatPtr(300.0)

	// Session 2: higher distance, higher sprints, higher everything (the record)
	s2 := validSessionBody()
	s2.StartedAt = start.Add(24 * time.Hour)
	s2.EndedAt = start.Add(24*time.Hour + 90*time.Minute)
	s2.DurationS = 5400
	s2.DistanceM = 10500
	s2.Sprints = 22
	s2.HRAvg = intPtr(145)
	s2.HRMax = intPtr(185)
	speed2 := 28.5
	s2.SpeedMaxKMH = &speed2
	s2.Intensity = floatPtr(78.0)
	s2.CaloriesKcal = floatPtr(520.0)

	for _, body := range []createSessionRequest{s1, s2} {
		rec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create status = %d; body = %s", rec.Code, rec.Body.String())
		}
	}

	rec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/records", "Bearer "+token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var pr session.PersonalRecords
	if err := json.NewDecoder(rec.Body).Decode(&pr); err != nil {
		t.Fatalf("decode: %v", err)
	}

	expect := map[string]float64{
		"distance_m":    10500,
		"duration_s":    5400,
		"speed_max_kmh": 28.5,
		"sprints":       22,
		"intensity":     78,
	}
	for _, r := range pr.Records {
		want, ok := expect[r.Metric]
		if !ok {
			continue
		}
		if r.Value != want {
			t.Errorf("record %s = %f, want %f", r.Metric, r.Value, want)
		}
		if r.SessionID == "" {
			t.Errorf("record %s has empty session_id", r.Metric)
		}
	}
}

func intPtr(v int) *int { return &v }
func floatPtr(v float64) *float64 { return &v }
