package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/MGeovany/rivalo-server/internal/session"
)

func TestGetRivalries_WithOpponents(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))

	start := time.Date(2026, 6, 1, 18, 0, 0, 0, time.UTC)
	opponent := "FC Barcelona"
	win := session.OutcomeWin
	draw := session.OutcomeDraw

	// Create 3 sessions vs same opponent.
	rec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	if rec.Code != http.StatusCreated {
		t.Fatalf("create #1 status = %d", rec.Code)
	}
	s1 := struct{ ID string `json:"id"` }{}
	json.NewDecoder(rec.Body).Decode(&s1)

	_ = doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+s1.ID, "Bearer "+token, map[string]any{
		"opponent": opponent,
		"outcome":  win,
	})

	// Second session
	body2 := validSessionBody()
	body2.StartedAt = start.AddDate(0, 0, -7)
	body2.EndedAt = start.AddDate(0, 0, -7).Add(90 * time.Minute)
	rec2 := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body2)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("create #2 status = %d", rec2.Code)
	}
	var s2 struct{ ID string `json:"id"` }
	json.NewDecoder(rec2.Body).Decode(&s2)
	_ = doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+s2.ID, "Bearer "+token, map[string]any{
		"opponent": opponent,
		"outcome":  draw,
	})

	// Third session — different rival so it should NOT appear (below threshold).
	body3 := validSessionBody()
	body3.StartedAt = start.AddDate(0, 0, -3)
	body3.EndedAt = start.AddDate(0, 0, -3).Add(90 * time.Minute)
	rec3 := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body3)
	if rec3.Code != http.StatusCreated {
		t.Fatalf("create #3 status = %d", rec3.Code)
	}
	var s3 struct{ ID string `json:"id"` }
	json.NewDecoder(rec3.Body).Decode(&s3)
	_ = doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+s3.ID, "Bearer "+token, map[string]any{
		"opponent": "One-off FC",
		"outcome":  win,
	})

	resp := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/rivalries", "Bearer "+token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", resp.Code, resp.Body.String())
	}

	var rivals []session.Rivalry
	if err := json.NewDecoder(resp.Body).Decode(&rivals); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rivals) != 1 {
		t.Fatalf("got %d rivalries, want 1 (One-off FC below threshold)", len(rivals))
	}
	if rivals[0].Opponent != opponent {
		t.Fatalf("opponent = %q, want %q", rivals[0].Opponent, opponent)
	}
	if rivals[0].MatchCount != 2 || rivals[0].Wins != 1 || rivals[0].Draws != 1 || rivals[0].Losses != 0 {
		t.Fatalf("W/D/L = %d/%d/%d count=%d, want 1/1/0/2", rivals[0].Wins, rivals[0].Draws, rivals[0].Losses, rivals[0].MatchCount)
	}
}

func TestGetRivalries_NoOpponents_Empty(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))

	resp := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/rivalries", "Bearer "+token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	var rivals []session.Rivalry
	if err := json.NewDecoder(resp.Body).Decode(&rivals); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rivals) != 0 {
		t.Fatalf("got %d rivalries, want 0", len(rivals))
	}
}

func TestGetRivalries_Unauthenticated_401(t *testing.T) {
	resp := doRequest(t, sessionDeps(newFakeSessionStore()), http.MethodGet, "/v1/rivalries", "", nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.Code)
	}
}
