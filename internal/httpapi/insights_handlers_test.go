package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/MGeovany/rivalo-server/internal/session"
)

func TestGetInsights_NoSessions_200(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-empty", time.Now().Add(time.Hour))
	rec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/insights", "Bearer "+token, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var ins session.SessionInsights
	if err := json.NewDecoder(rec.Body).Decode(&ins); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ins.Totals.SessionCount != 0 {
		t.Fatalf("session_count = %d, want 0", ins.Totals.SessionCount)
	}
}

func TestGetInsights_WithSessions_200(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-insights", time.Now().Add(time.Hour))
	start := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)

	mt := "11-a-side"
	sf := "Artificial turf"
	pos := "Midfielder"

	body := validSessionBody()
	body.StartedAt = start
	body.EndedAt = start.Add(90 * time.Minute)
	body.DurationS = 5400
	body.DistanceM = 10000
	body.Sprints = 15
	body.HRAvg = intPtr(140)
	body.HRMax = intPtr(175)
	speed := 25.0
	body.SpeedMaxKMH = &speed
	body.Intensity = floatPtr(70.0)
	body.CaloriesKcal = floatPtr(500.0)

	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body = %s", createRec.Code, createRec.Body.String())
	}
	var created session.Session
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}

	patchBody := patchSessionRequest{MatchType: &mt, Surface: &sf, Position: &pos}
	patchRec := doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+created.ID, "Bearer "+token, patchBody)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch status = %d; body = %s", patchRec.Code, patchRec.Body.String())
	}

	body2 := validSessionBody()
	body2.StartedAt = start.Add(48 * time.Hour)
	body2.EndedAt = start.Add(48*time.Hour + 60*time.Minute)
	body2.DurationS = 3600
	body2.DistanceM = 6000
	body2.Sprints = 8
	body2.HRAvg = intPtr(130)
	body2.HRMax = intPtr(160)
	speed2 := 20.0
	body2.SpeedMaxKMH = &speed2
	body2.Intensity = floatPtr(55.0)
	mt2 := "5-a-side"
	pos2 := "Forward"
	createRec2 := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body2)
	if createRec2.Code != http.StatusCreated {
		t.Fatalf("create2 status = %d; body = %s", createRec2.Code, createRec2.Body.String())
	}
	var created2 session.Session
	if err := json.NewDecoder(createRec2.Body).Decode(&created2); err != nil {
		t.Fatalf("decode created2: %v", err)
	}
	patchBody2 := patchSessionRequest{MatchType: &mt2, Position: &pos2}
	patchRec2 := doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+created2.ID, "Bearer "+token, patchBody2)
	if patchRec2.Code != http.StatusOK {
		t.Fatalf("patch2 status = %d; body = %s", patchRec2.Code, patchRec2.Body.String())
	}

	rec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/insights", "Bearer "+token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var ins session.SessionInsights
	if err := json.NewDecoder(rec.Body).Decode(&ins); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if ins.Totals.SessionCount != 2 {
		t.Errorf("session_count = %d, want 2", ins.Totals.SessionCount)
	}
	if ins.Totals.TotalDistanceM != 16000 {
		t.Errorf("total_distance = %f, want 16000", ins.Totals.TotalDistanceM)
	}

	if ins.Averages.DistancePerMatch == nil || *ins.Averages.DistancePerMatch != 8000 {
		t.Errorf("avg_distance = %v, want 8000", ins.Averages.DistancePerMatch)
	}

	if len(ins.ByMatchType) != 2 {
		t.Errorf("by_match_type count = %d, want 2", len(ins.ByMatchType))
	}
	if len(ins.ByPosition) != 2 {
		t.Errorf("by_position count = %d, want 2", len(ins.ByPosition))
	}

	_ = created
	_ = created2
}
