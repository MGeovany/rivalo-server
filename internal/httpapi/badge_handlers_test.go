package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/MGeovany/rivalo-server/internal/auth"
	"github.com/MGeovany/rivalo-server/internal/badge"
)

type fakeBadgeStore struct {
	mu     sync.Mutex
	earned map[string]map[string]time.Time
	grants int
}

func newFakeBadgeStore() *fakeBadgeStore {
	return &fakeBadgeStore{earned: map[string]map[string]time.Time{}}
}

func (f *fakeBadgeStore) Earned(_ context.Context, userID string) (map[string]time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]time.Time{}
	for k, v := range f.earned[userID] {
		out[k] = v
	}
	return out, nil
}

func (f *fakeBadgeStore) Grant(_ context.Context, userID, key string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.earned[userID] == nil {
		f.earned[userID] = map[string]time.Time{}
	}
	if _, ok := f.earned[userID][key]; ok {
		return nil // idempotent
	}
	f.earned[userID][key] = at
	f.grants++
	return nil
}

func TestGetBadges_GrantsOnceAndReturnsProgress(t *testing.T) {
	ss := newFakeSessionStore()
	bs := newFakeBadgeStore()
	d := Deps{Sessions: ss, Badges: bs, Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))

	// One session → first_match becomes earnable.
	if rec := doRequest(t, d, http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody()); rec.Code != http.StatusCreated {
		t.Fatalf("create session status = %d", rec.Code)
	}

	first := doRequest(t, d, http.MethodGet, "/v1/badges", "Bearer "+token, nil)
	if first.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", first.Code, first.Body.String())
	}
	var resp badgesResponse
	if err := json.NewDecoder(first.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var firstMatch *badge.Badge
	for i := range resp.Badges {
		if resp.Badges[i].Key == "first_match" {
			firstMatch = &resp.Badges[i]
		}
	}
	if firstMatch == nil || !firstMatch.Earned {
		t.Fatalf("first_match should be earned, got %+v", firstMatch)
	}
	grantsAfterFirst := bs.grants
	if grantsAfterFirst < 1 {
		t.Fatalf("expected at least one grant, got %d", grantsAfterFirst)
	}

	// Second fetch must not re-grant already-earned badges.
	_ = doRequest(t, d, http.MethodGet, "/v1/badges", "Bearer "+token, nil)
	if bs.grants != grantsAfterFirst {
		t.Fatalf("grants changed on second fetch: %d → %d (should be idempotent)", grantsAfterFirst, bs.grants)
	}
}
