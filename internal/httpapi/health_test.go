package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// stubPinger is a test double for the Pinger interface.
type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error { return s.err }

func TestHealth_NoDatabase(t *testing.T) {
	rec := doHealthRequest(t, NewRouter(Deps{DB: nil}))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeHealth(t, rec)
	if body.Status != "ok" {
		t.Errorf("status = %q, want %q", body.Status, "ok")
	}
	if body.Database != "disabled" {
		t.Errorf("database = %q, want %q", body.Database, "disabled")
	}
}

func TestHealth_DatabaseReachable(t *testing.T) {
	rec := doHealthRequest(t, NewRouter(Deps{DB: stubPinger{err: nil}}))

	body := decodeHealth(t, rec)
	if body.Database != "ok" {
		t.Errorf("database = %q, want %q", body.Database, "ok")
	}
}

func TestHealth_DatabaseUnreachable(t *testing.T) {
	rec := doHealthRequest(t, NewRouter(Deps{DB: stubPinger{err: errors.New("boom")}}))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (health stays live even if db is down)", rec.Code, http.StatusOK)
	}
	body := decodeHealth(t, rec)
	if body.Database != "unreachable" {
		t.Errorf("database = %q, want %q", body.Database, "unreachable")
	}
}

func doHealthRequest(t *testing.T, h http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeHealth(t *testing.T, rec *httptest.ResponseRecorder) healthResponse {
	t.Helper()
	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}
