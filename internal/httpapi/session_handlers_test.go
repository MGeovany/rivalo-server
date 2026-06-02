package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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
		Mode:         n.Mode,
		HalftimeOffsetS: n.HalftimeOffsetS,
		Samples:      n.Samples,
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

func (f *fakeSessionStore) Update(_ context.Context, userID, id string, u session.Update) (session.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, s := range f.items[userID] {
		if s.ID == id {
			s.StartedAt = u.StartedAt
			s.EndedAt = u.EndedAt
			s.DurationS = u.DurationS
			s.DistanceM = u.DistanceM
			s.HRAvg = u.HRAvg
			s.HRMax = u.HRMax
			s.SpeedMaxKMH = u.SpeedMaxKMH
			s.Sprints = u.Sprints
			s.Intensity = u.Intensity
			s.CaloriesKcal = u.CaloriesKcal
			f.items[userID][i] = s
			return s, nil
		}
	}
	return session.Session{}, session.ErrNotFound
}

func (f *fakeSessionStore) Delete(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.items[userID]
	for i, s := range list {
		if s.ID == id {
			f.items[userID] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return session.ErrNotFound
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

func TestCreateSession_OversizedBody_400(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	body := map[string]string{"source": "manual", "junk": strings.Repeat("a", 2<<20)} // ~2 MiB
	rec := doRequest(t, sessionDeps(newFakeSessionStore()), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (oversized body must be rejected)", rec.Code)
	}
}

func TestSession_WithSamples_RoundTrip(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	hr := 150
	body := validSessionBody()
	body.Samples = []sampleRequest{
		{TOffsetS: 0, HR: &hr},
		{TOffsetS: 10, HR: &hr},
	}

	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body = %s", createRec.Code, createRec.Body.String())
	}
	var created session.Session
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if len(created.Samples) != 2 {
		t.Fatalf("created samples = %d, want 2", len(created.Samples))
	}

	getRec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/"+created.ID, "Bearer "+token, nil)
	var detail session.Session
	if err := json.NewDecoder(getRec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if len(detail.Samples) != 2 || detail.Samples[0].HR == nil || *detail.Samples[0].HR != 150 {
		t.Errorf("detail samples not round-tripped: %+v", detail.Samples)
	}
}

func TestCreateSession_StructuredWithHalves(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	hr := 150
	half1, half2 := 1, 2
	offset := 2700
	body := validSessionBody()
	body.Source = session.SourceWatch
	body.Mode = session.ModeStructured
	body.HalftimeOffsetS = &offset
	body.Samples = []sampleRequest{
		{TOffsetS: 0, HR: &hr, Half: &half1},
		{TOffsetS: 3000, HR: &hr, Half: &half2},
	}

	rec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var created session.Session
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Mode != session.ModeStructured {
		t.Errorf("mode = %q, want structured", created.Mode)
	}
	if created.HalftimeOffsetS == nil || *created.HalftimeOffsetS != offset {
		t.Errorf("halftime offset not persisted: %+v", created.HalftimeOffsetS)
	}
	if len(created.Samples) != 2 || created.Samples[0].Half == nil || *created.Samples[0].Half != 1 {
		t.Errorf("sample half not persisted: %+v", created.Samples)
	}
}

func TestCreateSession_InvalidMode_400(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	body := validSessionBody()
	body.Mode = "tournament"
	rec := doRequest(t, sessionDeps(newFakeSessionStore()), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateSession_HalftimeOnNonStructured_400(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	offset := 1000
	body := validSessionBody() // mode defaults to quick
	body.HalftimeOffsetS = &offset
	rec := doRequest(t, sessionDeps(newFakeSessionStore()), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (halftime only for structured)", rec.Code)
	}
}

func TestCreateSession_DefaultsToQuick(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	var created session.Session
	_ = json.NewDecoder(rec.Body).Decode(&created)
	if created.Mode != session.ModeQuick {
		t.Errorf("mode = %q, want quick (default)", created.Mode)
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
