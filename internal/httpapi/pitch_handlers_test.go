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
	"github.com/MGeovany/rivalo-server/internal/pitch"
	"github.com/MGeovany/rivalo-server/internal/session"
)

// fakePitchStore is an in-memory pitch.Store for handler tests.
type fakePitchStore struct {
	mu    sync.Mutex
	items map[string][]pitch.Pitch
	seq   int
}

func newFakePitchStore() *fakePitchStore {
	return &fakePitchStore{items: map[string][]pitch.Pitch{}}
}

func (f *fakePitchStore) Create(_ context.Context, userID string, n pitch.NewPitch) (pitch.Pitch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	now := time.Now()
	p := pitch.Pitch{
		ID:                "pitch-" + strconv.Itoa(f.seq),
		UserID:            userID,
		Name:              n.Name,
		Latitude:          n.Latitude,
		Longitude:         n.Longitude,
		Type:              n.Type,
		Surface:           n.Surface,
		LengthM:           n.LengthM,
		WidthM:            n.WidthM,
		MeasurementMethod: n.MeasurementMethod,
		Indoor:            n.Indoor,
		Notes:             n.Notes,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	f.items[userID] = append(f.items[userID], p)
	return p, nil
}

func (f *fakePitchStore) Get(_ context.Context, userID, id string) (pitch.Pitch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.items[userID] {
		if p.ID == id {
			return p, nil
		}
	}
	return pitch.Pitch{}, pitch.ErrNotFound
}

func (f *fakePitchStore) List(_ context.Context, userID string) ([]pitch.Pitch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]pitch.Pitch(nil), f.items[userID]...), nil
}

func (f *fakePitchStore) Update(_ context.Context, userID, id string, u pitch.PitchUpdate) (pitch.Pitch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, p := range f.items[userID] {
		if p.ID == id {
			if u.Name != nil {
				p.Name = *u.Name
			}
			if u.Latitude != nil {
				p.Latitude = u.Latitude
			}
			if u.Longitude != nil {
				p.Longitude = u.Longitude
			}
			if u.Type != nil {
				p.Type = u.Type
			}
			if u.Surface != nil {
				p.Surface = u.Surface
			}
			if u.LengthM != nil {
				p.LengthM = u.LengthM
			}
			if u.WidthM != nil {
				p.WidthM = u.WidthM
			}
			if u.MeasurementMethod != nil {
				p.MeasurementMethod = u.MeasurementMethod
			}
			if u.Indoor != nil {
				p.Indoor = u.Indoor
			}
			if u.Notes != nil {
				p.Notes = u.Notes
			}
			f.items[userID][i] = p
			return p, nil
		}
	}
	return pitch.Pitch{}, pitch.ErrNotFound
}

func (f *fakePitchStore) Delete(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.items[userID]
	for i, p := range list {
		if p.ID == id {
			f.items[userID] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return pitch.ErrNotFound
}

func (f *fakePitchStore) OwnedByUser(_ context.Context, userID, id string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.items[userID] {
		if p.ID == id {
			return true, nil
		}
	}
	return false, nil
}

func pitchDeps(ps *fakePitchStore) Deps {
	return Deps{Pitches: ps, Verifier: auth.NewVerifier(testSecret)}
}

func validPitchBody() createPitchRequest {
	t := "5-a-side"
	return createPitchRequest{
		Name:  "Home field",
		Type:  &t,
	}
}

func TestCreatePitch_Valid_201(t *testing.T) {
	store := newFakePitchStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, pitchDeps(store), http.MethodPost, "/v1/pitches", "Bearer "+token, validPitchBody())
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var p pitch.Pitch
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p.UserID != "user-1" || p.Name != "Home field" || p.ID == "" {
		t.Errorf("unexpected pitch: %+v", p)
	}
}

func TestCreatePitch_DetailFields_RoundTrip(t *testing.T) {
	ps := newFakePitchStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	body := validPitchBody()
	indoor := true
	notes := "Lights until 10pm, gate code 1234"
	body.Indoor = &indoor
	body.Notes = &notes

	rec := doRequest(t, pitchDeps(ps), http.MethodPost, "/v1/pitches", "Bearer "+token, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var created pitch.Pitch
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Indoor == nil || !*created.Indoor {
		t.Errorf("indoor not persisted: %+v", created.Indoor)
	}
	if created.Notes == nil || *created.Notes != notes {
		t.Errorf("notes not persisted: %+v", created.Notes)
	}
}

func TestCreatePitch_NoName_400(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	body := createPitchRequest{Name: ""}
	rec := doRequest(t, pitchDeps(newFakePitchStore()), http.MethodPost, "/v1/pitches", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreatePitch_InvalidType_400(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	bad := "invalid"
	body := createPitchRequest{Name: "Test", Type: &bad}
	rec := doRequest(t, pitchDeps(newFakePitchStore()), http.MethodPost, "/v1/pitches", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreatePitch_InvalidSurface_400(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	bad := "plastic"
	body := createPitchRequest{Name: "Test", Surface: &bad}
	rec := doRequest(t, pitchDeps(newFakePitchStore()), http.MethodPost, "/v1/pitches", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestListPitches_OnlyOwn(t *testing.T) {
	store := newFakePitchStore()
	tokenA := signToken(t, testSecret, "user-a", time.Now().Add(time.Hour))
	tokenB := signToken(t, testSecret, "user-b", time.Now().Add(time.Hour))

	_ = doRequest(t, pitchDeps(store), http.MethodPost, "/v1/pitches", "Bearer "+tokenA, validPitchBody())
	_ = doRequest(t, pitchDeps(store), http.MethodPost, "/v1/pitches", "Bearer "+tokenB, validPitchBody())

	rec := doRequest(t, pitchDeps(store), http.MethodGet, "/v1/pitches", "Bearer "+tokenA, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var list []pitch.Pitch
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 1 || list[0].UserID != "user-a" {
		t.Errorf("expected only user-a's pitch, got %+v", list)
	}
}

func TestGetPitch_OwnerAndNotOwner(t *testing.T) {
	store := newFakePitchStore()
	tokenA := signToken(t, testSecret, "user-a", time.Now().Add(time.Hour))
	tokenB := signToken(t, testSecret, "user-b", time.Now().Add(time.Hour))

	createRec := doRequest(t, pitchDeps(store), http.MethodPost, "/v1/pitches", "Bearer "+tokenA, validPitchBody())
	var created pitch.Pitch
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	ownerRec := doRequest(t, pitchDeps(store), http.MethodGet, "/v1/pitches/"+created.ID, "Bearer "+tokenA, nil)
	if ownerRec.Code != http.StatusOK {
		t.Fatalf("owner status = %d, want 200", ownerRec.Code)
	}

	otherRec := doRequest(t, pitchDeps(store), http.MethodGet, "/v1/pitches/"+created.ID, "Bearer "+tokenB, nil)
	if otherRec.Code != http.StatusNotFound {
		t.Fatalf("non-owner status = %d, want 404", otherRec.Code)
	}

	missingRec := doRequest(t, pitchDeps(store), http.MethodGet, "/v1/pitches/nope", "Bearer "+tokenA, nil)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", missingRec.Code)
	}
}

func TestUpdatePitch_Valid(t *testing.T) {
	store := newFakePitchStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	createRec := doRequest(t, pitchDeps(store), http.MethodPost, "/v1/pitches", "Bearer "+token, validPitchBody())
	var created pitch.Pitch
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	newName := "Away field"
	s := "Artificial turf"
	body := updatePitchRequest{Name: &newName, Surface: &s}
	rec := doRequest(t, pitchDeps(store), http.MethodPut, "/v1/pitches/"+created.ID, "Bearer "+token, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var updated pitch.Pitch
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Name != "Away field" {
		t.Errorf("name = %q, want 'Away field'", updated.Name)
	}
	if updated.Surface == nil || *updated.Surface != "Artificial turf" {
		t.Errorf("surface not updated: %+v", updated.Surface)
	}
	// Type should remain unchanged.
	if updated.Type == nil || *updated.Type != "5-a-side" {
		t.Errorf("type should have been preserved: %+v", updated.Type)
	}
}

func TestUpdatePitch_NotFound_404(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	name := "Nonexistent"
	body := updatePitchRequest{Name: &name}
	rec := doRequest(t, pitchDeps(newFakePitchStore()), http.MethodPut, "/v1/pitches/nope", "Bearer "+token, body)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDeletePitch_Valid(t *testing.T) {
	store := newFakePitchStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	createRec := doRequest(t, pitchDeps(store), http.MethodPost, "/v1/pitches", "Bearer "+token, validPitchBody())
	var created pitch.Pitch
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	rec := doRequest(t, pitchDeps(store), http.MethodDelete, "/v1/pitches/"+created.ID, "Bearer "+token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}

	// Second delete should 404.
	rec2 := doRequest(t, pitchDeps(store), http.MethodDelete, "/v1/pitches/"+created.ID, "Bearer "+token, nil)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want 404", rec2.Code)
	}
}

func TestDeletePitch_NotFound_404(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, pitchDeps(newFakePitchStore()), http.MethodDelete, "/v1/pitches/nope", "Bearer "+token, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestCreateSession_InvalidPitchID_400(t *testing.T) {
	ps := newFakePitchStore()
	ss := newFakeSessionStore()
	d := Deps{Sessions: ss, Pitches: ps, Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	badID := "nonexistent-pitch"
	body := validSessionBody()
	body.PitchID = &badID

	rec := doRequest(t, d, http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateSession_ValidPitchID(t *testing.T) {
	ps := newFakePitchStore()
	ss := newFakeSessionStore()
	d := Deps{Sessions: ss, Pitches: ps, Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))

	// Create a pitch first.
	createPitchRec := doRequest(t, d, http.MethodPost, "/v1/pitches", "Bearer "+token, validPitchBody())
	var pitchCreated pitch.Pitch
	_ = json.NewDecoder(createPitchRec.Body).Decode(&pitchCreated)

	// Create a session with that pitch.
	body := validSessionBody()
	body.PitchID = &pitchCreated.ID
	rec := doRequest(t, d, http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var created session.Session
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.PitchID == nil || *created.PitchID != pitchCreated.ID {
		t.Errorf("pitch_id not associated: %+v", created.PitchID)
	}
}

func TestPatchSessionContext_InvalidPitchID_400(t *testing.T) {
	ps := newFakePitchStore()
	ss := newFakeSessionStore()
	d := Deps{Sessions: ss, Pitches: ps, Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))

	createRec := doRequest(t, d, http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	var sessCreated session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&sessCreated)

	badID := "nonexistent"
	body := patchSessionRequest{PitchID: &badID}
	rec := doRequest(t, d, http.MethodPatch, "/v1/sessions/"+sessCreated.ID, "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}
