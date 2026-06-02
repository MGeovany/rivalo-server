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
	"github.com/MGeovany/rivalo-server/internal/goal"
)

type fakeGoalStore struct {
	mu    sync.Mutex
	items map[string][]goal.Goal
	seq   int
}

func newFakeGoalStore() *fakeGoalStore {
	return &fakeGoalStore{items: map[string][]goal.Goal{}}
}

func (f *fakeGoalStore) Create(_ context.Context, userID string, n goal.NewGoal) (goal.Goal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	g := goal.Goal{
		ID:        "goal-" + strconv.Itoa(f.seq),
		UserID:    userID,
		Metric:    n.Metric,
		Period:    n.Period,
		Target:    n.Target,
		CreatedAt: time.Now().UTC(),
	}
	f.items[userID] = append(f.items[userID], g)
	return g, nil
}

func (f *fakeGoalStore) List(_ context.Context, userID string) ([]goal.Goal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []goal.Goal
	for _, g := range f.items[userID] {
		if !g.Archived {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *fakeGoalStore) Get(_ context.Context, userID, id string) (goal.Goal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, g := range f.items[userID] {
		if g.ID == id {
			return g, nil
		}
	}
	return goal.Goal{}, goal.ErrNotFound
}

func (f *fakeGoalStore) Update(_ context.Context, userID, id string, u goal.GoalUpdate) (goal.Goal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, g := range f.items[userID] {
		if g.ID == id {
			if u.Metric != nil { g.Metric = *u.Metric }
			if u.Period != nil { g.Period = *u.Period }
			if u.Target != nil { g.Target = *u.Target }
			if u.Archived != nil { g.Archived = *u.Archived }
			f.items[userID][i] = g
			return g, nil
		}
	}
	return goal.Goal{}, goal.ErrNotFound
}

func (f *fakeGoalStore) Delete(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.items[userID]
	for i, g := range list {
		if g.ID == id {
			f.items[userID] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return goal.ErrNotFound
}

func (f *fakeGoalStore) Achieve(_ context.Context, userID, id string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, g := range f.items[userID] {
		if g.ID == id && g.AchievedAt == nil {
			g.AchievedAt = &at
			f.items[userID][i] = g
			return nil
		}
	}
	return goal.ErrNotFound
}

func TestCreateGoal_Valid_201(t *testing.T) {
	gs := newFakeGoalStore()
	ss := newFakeSessionStore()
	d := Deps{Goals: gs, Sessions: ss, Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))

	body := map[string]any{"metric": "distance", "period": "week", "target": 15000}
	rec := doRequest(t, d, http.MethodPost, "/v1/goals", "Bearer "+token, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}

	var created goal.Goal
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Metric != "distance" || created.Period != "week" || created.Target != 15000 {
		t.Errorf("unexpected goal: %+v", created)
	}
}

func TestCreateGoal_InvalidMetric_400(t *testing.T) {
	d := Deps{Goals: newFakeGoalStore(), Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, d, http.MethodPost, "/v1/goals", "Bearer "+token, map[string]any{
		"metric": "invalid", "period": "week", "target": 10,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateGoal_InvalidPeriod_400(t *testing.T) {
	d := Deps{Goals: newFakeGoalStore(), Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, d, http.MethodPost, "/v1/goals", "Bearer "+token, map[string]any{
		"metric": "matches", "period": "year", "target": 10,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateGoal_NegativeTarget_400(t *testing.T) {
	d := Deps{Goals: newFakeGoalStore(), Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, d, http.MethodPost, "/v1/goals", "Bearer "+token, map[string]any{
		"metric": "matches", "period": "week", "target": -5,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestListGoals_Empty(t *testing.T) {
	d := Deps{Goals: newFakeGoalStore(), Sessions: newFakeSessionStore(), Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, d, http.MethodGet, "/v1/goals", "Bearer "+token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var goals []goal.Goal
	if err := json.NewDecoder(rec.Body).Decode(&goals); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(goals) != 0 {
		t.Fatalf("got %d goals, want 0", len(goals))
	}
}

func TestGetGoal_NotFound_404(t *testing.T) {
	d := Deps{Goals: newFakeGoalStore(), Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, d, http.MethodGet, "/v1/goals/nonexistent", "Bearer "+token, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGoalCRUD_FullCycle(t *testing.T) {
	gs := newFakeGoalStore()
	ss := newFakeSessionStore()
	d := Deps{Goals: gs, Sessions: ss, Verifier: auth.NewVerifier(testSecret)}
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))

	// Create.
	createResp := doRequest(t, d, http.MethodPost, "/v1/goals", "Bearer "+token, map[string]any{
		"metric": "matches", "period": "week", "target": 3,
	})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status = %d", createResp.Code)
	}
	var created goal.Goal
	json.NewDecoder(createResp.Body).Decode(&created)

	// List.
	listResp := doRequest(t, d, http.MethodGet, "/v1/goals", "Bearer "+token, nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d", listResp.Code)
	}
	var list []goal.Goal
	json.NewDecoder(listResp.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	// Get.
	getResp := doRequest(t, d, http.MethodGet, "/v1/goals/"+created.ID, "Bearer "+token, nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d", getResp.Code)
	}
	var fetched goal.Goal
	json.NewDecoder(getResp.Body).Decode(&fetched)
	if fetched.Metric != "matches" {
		t.Fatalf("fetched metric = %q, want matches", fetched.Metric)
	}

	// Update (archive).
	updateResp := doRequest(t, d, http.MethodPatch, "/v1/goals/"+created.ID, "Bearer "+token, map[string]any{
		"archived": true,
	})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status = %d", updateResp.Code)
	}
	var updated goal.Goal
	json.NewDecoder(updateResp.Body).Decode(&updated)
	if !updated.Archived {
		t.Fatal("goal should be archived after update")
	}

	// List should now be empty (archived goals excluded).
	list2 := doRequest(t, d, http.MethodGet, "/v1/goals", "Bearer "+token, nil)
	var list2Goals []goal.Goal
	json.NewDecoder(list2.Body).Decode(&list2Goals)
	if len(list2Goals) != 0 {
		t.Fatalf("list after archive = %d, want 0", len(list2Goals))
	}

	// Delete.
	delResp := doRequest(t, d, http.MethodDelete, "/v1/goals/"+created.ID, "Bearer "+token, nil)
	if delResp.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", delResp.Code)
	}

	// Get after delete should 404.
	getDelResp := doRequest(t, d, http.MethodGet, "/v1/goals/"+created.ID, "Bearer "+token, nil)
	if getDelResp.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", getDelResp.Code)
	}
}
