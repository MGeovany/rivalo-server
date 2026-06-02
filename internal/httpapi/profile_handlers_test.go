package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/MGeovany/rivalo-server/internal/auth"
	"github.com/MGeovany/rivalo-server/internal/profile"
)

const testSecret = "test-secret-value"

// fakeStore is an in-memory profile.Store for handler tests.
type fakeStore struct {
	profiles map[string]profile.Profile
	fail     bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{profiles: map[string]profile.Profile{}}
}

func (f *fakeStore) GetOrCreate(_ context.Context, id string) (profile.Profile, error) {
	if f.fail {
		return profile.Profile{}, errors.New("store failure")
	}
	p, ok := f.profiles[id]
	if !ok {
		p = profile.Profile{ID: id}
		f.profiles[id] = p
	}
	return p, nil
}

func (f *fakeStore) Update(_ context.Context, id string, u profile.Update) (profile.Profile, error) {
	if f.fail {
		return profile.Profile{}, errors.New("store failure")
	}
	p := profile.Profile{
		ID:                id,
		DisplayName:       u.DisplayName,
		PreferredPosition: u.PreferredPosition,
		HeightCM:          u.HeightCM,
		WeightKG:          u.WeightKG,
		BirthYear:         u.BirthYear,
	}
	f.profiles[id] = p
	return p, nil
}

func testDeps(store profile.Store) Deps {
	return Deps{Profiles: store, Verifier: auth.NewVerifier(testSecret)}
}

func signToken(t *testing.T, secret, sub string, exp time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{"sub": sub, "exp": exp.Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func doRequest(t *testing.T, d Deps, method, target, authHeader string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	NewRouter(d).ServeHTTP(rec, req)
	return rec
}

func TestGetMe_MissingToken_401(t *testing.T) {
	rec := doRequest(t, testDeps(newFakeStore()), http.MethodGet, "/v1/me", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestGetMe_InvalidSignature_401(t *testing.T) {
	token := signToken(t, "wrong-secret", "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, testDeps(newFakeStore()), http.MethodGet, "/v1/me", "Bearer "+token, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestGetMe_ExpiredToken_401(t *testing.T) {
	token := signToken(t, testSecret, "user-1", time.Now().Add(-time.Hour))
	rec := doRequest(t, testDeps(newFakeStore()), http.MethodGet, "/v1/me", "Bearer "+token, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestGetMe_ValidToken_CreatesProfile(t *testing.T) {
	store := newFakeStore()
	token := signToken(t, testSecret, "user-42", time.Now().Add(time.Hour))
	rec := doRequest(t, testDeps(store), http.MethodGet, "/v1/me", "Bearer "+token, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var p profile.Profile
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p.ID != "user-42" {
		t.Errorf("id = %q, want user-42", p.ID)
	}
	if _, ok := store.profiles["user-42"]; !ok {
		t.Errorf("profile was not created in the store")
	}
}

func TestAuth_NotConfigured_503(t *testing.T) {
	d := Deps{Profiles: newFakeStore(), Verifier: auth.NewVerifier("")}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, d, http.MethodGet, "/v1/me", "Bearer "+token, nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestUpdateMe_Valid(t *testing.T) {
	store := newFakeStore()
	token := signToken(t, testSecret, "user-7", time.Now().Add(time.Hour))
	pos := "midfielder"
	height := 180
	weight := 75.5
	body := updateProfileRequest{
		DisplayName:       "Leo",
		PreferredPosition: &pos,
		HeightCM:          &height,
		WeightKG:          &weight,
	}

	rec := doRequest(t, testDeps(store), http.MethodPut, "/v1/me", "Bearer "+token, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var p profile.Profile
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p.DisplayName != "Leo" || p.PreferredPosition == nil || *p.PreferredPosition != "midfielder" {
		t.Errorf("unexpected profile: %+v", p)
	}
}

func TestUpdateMe_MissingDisplayName_400(t *testing.T) {
	token := signToken(t, testSecret, "user-7", time.Now().Add(time.Hour))
	body := updateProfileRequest{DisplayName: "   "}
	rec := doRequest(t, testDeps(newFakeStore()), http.MethodPut, "/v1/me", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestUpdateMe_InvalidHeight_400(t *testing.T) {
	token := signToken(t, testSecret, "user-7", time.Now().Add(time.Hour))
	height := 5
	body := updateProfileRequest{DisplayName: "Leo", HeightCM: &height}
	rec := doRequest(t, testDeps(newFakeStore()), http.MethodPut, "/v1/me", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestUpdateMe_BirthYear_Valid(t *testing.T) {
	store := newFakeStore()
	token := signToken(t, testSecret, "user-8", time.Now().Add(time.Hour))
	by := 1995
	body := updateProfileRequest{DisplayName: "Vet", BirthYear: &by}
	rec := doRequest(t, testDeps(store), http.MethodPut, "/v1/me", "Bearer "+token, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var p profile.Profile
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p.BirthYear == nil || *p.BirthYear != 1995 {
		t.Errorf("birth_year = %+v, want 1995", p.BirthYear)
	}
}

func TestUpdateMe_BirthYear_OutOfRange_400(t *testing.T) {
	token := signToken(t, testSecret, "user-9", time.Now().Add(time.Hour))
	by := 1800
	body := updateProfileRequest{DisplayName: "Old", BirthYear: &by}
	rec := doRequest(t, testDeps(newFakeStore()), http.MethodPut, "/v1/me", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
