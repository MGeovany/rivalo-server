package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/MGeovany/rivalo-server/internal/auth"
	"github.com/MGeovany/rivalo-server/internal/session"
)

// fakeSessionStore is an in-memory session.Store for handler tests.
type fakeSessionStore struct {
	mu    sync.Mutex
	items map[string][]session.Session // keyed by user id
	seq   int
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{items: map[string][]session.Session{}}
}

func (f *fakeSessionStore) Create(_ context.Context, userID string, n session.New) (session.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	s := session.Session{
		ID:           "sess-" + strconv.Itoa(f.seq),
		UserID:       userID,
		StartedAt:    n.StartedAt,
		EndedAt:      n.EndedAt,
		DurationS:    n.DurationS,
		DistanceM:    n.DistanceM,
		HRAvg:        n.HRAvg,
		HRMax:        n.HRMax,
		SpeedMaxKMH:  n.SpeedMaxKMH,
		Sprints:      n.Sprints,
		Intensity:    n.Intensity,
		CaloriesKcal: n.CaloriesKcal,
		Source:       n.Source,
		CreatedAt:    n.StartedAt,
	}
	f.items[userID] = append(f.items[userID], s)
	return s, nil
}

func (f *fakeSessionStore) List(_ context.Context, userID string) ([]session.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]session.Session(nil), f.items[userID]...), nil
}

func (f *fakeSessionStore) Get(_ context.Context, userID, id string) (session.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.items[userID] {
		if s.ID == id {
			return s, nil
		}
	}
	return session.Session{}, session.ErrNotFound
}

func sessionDeps(store session.Store) Deps {
	return Deps{Sessions: store, Verifier: auth.NewVerifier(testSecret)}
}

func validSessionBody() createSessionRequest {
	start := time.Date(2026, 6, 1, 18, 0, 0, 0, time.UTC)
	return createSessionRequest{
		StartedAt: start,
		EndedAt:   start.Add(90 * time.Minute),
		DurationS: 5400,
		DistanceM: 8200,
		Sprints:   14,
		Source:    session.SourceManual,
	}
}

func TestCreateSession_MissingToken_401(t *testing.T) {
	rec := doRequest(t, sessionDeps(newFakeSessionStore()), http.MethodPost, "/v1/sessions", "", validSessionBody())
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestCreateSession_Valid_201(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var s session.Session
	if err := json.NewDecoder(rec.Body).Decode(&s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if s.UserID != "user-1" || s.Source != session.SourceManual || s.ID == "" {
		t.Errorf("unexpected session: %+v", s)
	}
}

func TestCreateSession_InvalidSource_400(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	body := validSessionBody()
	body.Source = "treadmill"
	rec := doRequest(t, sessionDeps(newFakeSessionStore()), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateSession_EndBeforeStart_400(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	body := validSessionBody()
	body.EndedAt = body.StartedAt.Add(-time.Minute)
	rec := doRequest(t, sessionDeps(newFakeSessionStore()), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestListSessions_OnlyOwn(t *testing.T) {
	store := newFakeSessionStore()
	tokenA := signToken(t, testSecret, "user-a", time.Now().Add(time.Hour))
	tokenB := signToken(t, testSecret, "user-b", time.Now().Add(time.Hour))

	_ = doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+tokenA, validSessionBody())
	_ = doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+tokenB, validSessionBody())

	rec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions", "Bearer "+tokenA, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var list []session.Session
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 1 || list[0].UserID != "user-a" {
		t.Errorf("expected only user-a's session, got %+v", list)
	}
}

func TestGetSession_OwnerAndNotOwner(t *testing.T) {
	store := newFakeSessionStore()
	tokenA := signToken(t, testSecret, "user-a", time.Now().Add(time.Hour))
	tokenB := signToken(t, testSecret, "user-b", time.Now().Add(time.Hour))

	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+tokenA, validSessionBody())
	var created session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	// Owner can fetch it.
	ownerRec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/"+created.ID, "Bearer "+tokenA, nil)
	if ownerRec.Code != http.StatusOK {
		t.Fatalf("owner status = %d, want 200", ownerRec.Code)
	}

	// A different user gets 404 (existence is not revealed).
	otherRec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/"+created.ID, "Bearer "+tokenB, nil)
	if otherRec.Code != http.StatusNotFound {
		t.Fatalf("non-owner status = %d, want 404", otherRec.Code)
	}

	// Missing id also 404.
	missingRec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/nope", "Bearer "+tokenA, nil)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", missingRec.Code)
	}
}
