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
		MatchRating:  n.MatchRating,
		PitchID:      n.PitchID,
		Samples:      n.Samples,
		Path:         n.Path,
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

func (f *fakeSessionStore) UpdateContext(_ context.Context, userID, id string, cu session.ContextUpdate) (session.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, s := range f.items[userID] {
		if s.ID == id {
			s.MatchType = cu.MatchType
			s.Surface = cu.Surface
			s.Position = cu.Position
			s.Result = cu.Result
			s.Feeling = cu.Feeling
			s.MatchTag = cu.MatchTag
			s.Opponent = cu.Opponent
			s.Outcome = cu.Outcome
			s.Score = cu.Score
			s.Competition = cu.Competition
			s.Goals = cu.Goals
			s.Assists = cu.Assists
			s.Notes = cu.Notes
			s.PitchID = cu.PitchID
			f.items[userID][i] = s
			return s, nil
		}
	}
	return session.Session{}, session.ErrNotFound
}

func (f *fakeSessionStore) GetPersonalRecords(_ context.Context, userID string) (session.PersonalRecords, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.items[userID]
	if len(list) == 0 {
		return session.PersonalRecords{Records: nil}, nil
	}

	var (
		bestDist     float64
		distSid      string
		distDate     time.Time
		bestDur      float64
		durSid       string
		durDate      time.Time
		bestSpeed    *float64
		speedSid     *string
		speedDate    *time.Time
		bestSprints  float64
		sprintSid    string
		sprintDate   time.Time
		bestInt      *float64
		intSid       *string
		intDate      *time.Time
		bestRating   *float64
		ratingSid    *string
		ratingDate   *time.Time
		bestHRMax    *float64
		hrSid        *string
		hrDate       *time.Time
		bestCals     *float64
		calsSid      *string
		calsDate     *time.Time
	)

	for _, s := range list {
		if s.DistanceM > bestDist {
			bestDist = s.DistanceM
			distSid = s.ID
			distDate = s.StartedAt
		}
		if float64(s.DurationS) > bestDur {
			bestDur = float64(s.DurationS)
			durSid = s.ID
			durDate = s.StartedAt
		}
		if s.SpeedMaxKMH != nil && (bestSpeed == nil || *s.SpeedMaxKMH > *bestSpeed) {
			v := *s.SpeedMaxKMH
			bestSpeed = &v
			speedSid = &s.ID
			speedDate = &s.StartedAt
		}
		if float64(s.Sprints) > bestSprints {
			bestSprints = float64(s.Sprints)
			sprintSid = s.ID
			sprintDate = s.StartedAt
		}
		if s.Intensity != nil && (bestInt == nil || *s.Intensity > *bestInt) {
			v := *s.Intensity
			bestInt = &v
			intSid = &s.ID
			intDate = &s.StartedAt
		}
		if s.MatchRating != nil && (bestRating == nil || *s.MatchRating > *bestRating) {
			v := *s.MatchRating
			bestRating = &v
			ratingSid = &s.ID
			ratingDate = &s.StartedAt
		}
		if s.HRMax != nil && (bestHRMax == nil || float64(*s.HRMax) > *bestHRMax) {
			v := float64(*s.HRMax)
			bestHRMax = &v
			hrSid = &s.ID
			hrDate = &s.StartedAt
		}
		if s.CaloriesKcal != nil && (bestCals == nil || *s.CaloriesKcal > *bestCals) {
			v := *s.CaloriesKcal
			bestCals = &v
			calsSid = &s.ID
			calsDate = &s.StartedAt
		}
	}

	records := make([]session.RecordEntry, 0, 9)

	add := func(metric string, value float64, sid string, date time.Time) {
		records = append(records, session.RecordEntry{Metric: metric, Value: value, SessionID: sid, StartedAt: date})
	}

	add("distance_m", bestDist, distSid, distDate)
	add("duration_s", bestDur, durSid, durDate)
	if bestSpeed != nil {
		add("speed_max_kmh", *bestSpeed, *speedSid, *speedDate)
	}
	add("sprints", bestSprints, sprintSid, sprintDate)
	if bestInt != nil {
		add("intensity", *bestInt, *intSid, *intDate)
	}
	if bestRating != nil {
		add("match_rating", *bestRating, *ratingSid, *ratingDate)
	}
	if bestHRMax != nil {
		add("hr_max", *bestHRMax, *hrSid, *hrDate)
	}
	if bestCals != nil {
		add("calories_kcal", *bestCals, *calsSid, *calsDate)
	}

	return session.PersonalRecords{Records: records}, nil
}

func (f *fakeSessionStore) GetInsights(_ context.Context, userID string) (session.SessionInsights, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.items[userID]
	if len(list) == 0 {
		return session.SessionInsights{}, nil
	}

	var ins session.SessionInsights
	ins.Totals.SessionCount = len(list)

	var distSum, durSum, sprintsSum, intSum, ratingSum float64
	var intCount, ratingCount, calsSum int
	for _, s := range list {
		ins.Totals.TotalDistanceM += s.DistanceM
		ins.Totals.TotalDurationS += s.DurationS
		distSum += s.DistanceM
		durSum += float64(s.DurationS)
		sprintsSum += float64(s.Sprints)
		if s.Intensity != nil {
			intSum += *s.Intensity
			intCount++
		}
		if s.MatchRating != nil {
			ratingSum += *s.MatchRating
			ratingCount++
		}
		if s.CaloriesKcal != nil {
			calsSum++
			ins.Totals.TotalCalories = floatPtr(float64(calsSum))
		}
	}

	n := float64(len(list))
	ins.Averages.DistancePerMatch = floatPtr(distSum / n)
	ins.Averages.DurationPerMatch = floatPtr(durSum / n)
	ins.Averages.SprintsPerMatch = floatPtr(sprintsSum / n)
	if intCount > 0 {
		ins.Averages.Intensity = floatPtr(intSum / float64(intCount))
	}
	if ratingCount > 0 {
		ins.Averages.MatchRating = floatPtr(ratingSum / float64(ratingCount))
	}

	ctxGroup := func(field string) []session.ContextGroup {
		type accum struct {
			count                                       int
			ratingSum, distanceSum, durationSum, intSum float64
			ratingN, intN                               int
		}
		m := map[string]*accum{}
		for _, s := range list {
			var val string
			switch field {
			case "match_type":
				if s.MatchType != nil {
					val = *s.MatchType
				}
			case "surface":
				if s.Surface != nil {
					val = *s.Surface
				}
			case "position":
				if s.Position != nil {
					val = *s.Position
				}
			}
			if val == "" {
				continue
			}
			if m[val] == nil {
				m[val] = &accum{}
			}
			a := m[val]
			a.count++
			a.distanceSum += s.DistanceM
			a.durationSum += float64(s.DurationS)
			if s.MatchRating != nil {
				a.ratingSum += *s.MatchRating
				a.ratingN++
			}
			if s.Intensity != nil {
				a.intSum += *s.Intensity
				a.intN++
			}
		}
		var groups []session.ContextGroup
		for val, a := range m {
			g := session.ContextGroup{Value: val, Count: a.count}
			g.AvgDistance = floatPtr(a.distanceSum / float64(a.count))
			g.AvgDurationS = floatPtr(a.durationSum / float64(a.count))
			if a.ratingN > 0 {
				g.AvgMatchRating = floatPtr(a.ratingSum / float64(a.ratingN))
			}
			if a.intN > 0 {
				g.AvgIntensity = floatPtr(a.intSum / float64(a.intN))
			}
			groups = append(groups, g)
		}
		return groups
	}

	ins.ByMatchType = ctxGroup("match_type")
	ins.BySurface = ctxGroup("surface")
	ins.ByPosition = ctxGroup("position")
	ins.Insights = session.BuildInsights(ins, ins.Averages.MatchRating)
	return ins, nil
}

func (f *fakeSessionStore) GetPositionInsights(_ context.Context, userID string) (session.PositionInsights, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	type accum struct {
		count                                  int
		distSum, sprintsSum, intSum, ratingSum float64
		durSum                                 float64
		intN, ratingN                          int
	}
	m := map[string]*accum{}
	for _, s := range f.items[userID] {
		if s.Position == nil || *s.Position == "" {
			continue
		}
		a := m[*s.Position]
		if a == nil {
			a = &accum{}
			m[*s.Position] = a
		}
		a.count++
		a.distSum += s.DistanceM
		a.sprintsSum += float64(s.Sprints)
		a.durSum += float64(s.DurationS)
		if s.Intensity != nil {
			a.intSum += *s.Intensity
			a.intN++
		}
		if s.MatchRating != nil {
			a.ratingSum += *s.MatchRating
			a.ratingN++
		}
	}

	var all []session.PositionStat
	for pos, a := range m {
		n := float64(a.count)
		p := session.PositionStat{
			Position:     pos,
			SessionCount: a.count,
			AvgDistanceM: floatPtr(a.distSum / n),
			AvgSprints:   floatPtr(a.sprintsSum / n),
			AvgDurationS: floatPtr(a.durSum / n),
		}
		if a.intN > 0 {
			p.AvgIntensity = floatPtr(a.intSum / float64(a.intN))
		}
		if a.ratingN > 0 {
			p.AvgMatchRating = floatPtr(a.ratingSum / float64(a.ratingN))
		}
		all = append(all, p)
	}
	return session.AssemblePositionInsights(all), nil
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

func TestPatchSessionContext_Valid(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	var created session.Session
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	mt := "11-a-side"
	sf := "Natural grass"
	pos := "Midfielder"
	res := "Won 3-1"
	feel := 4
	tag := "league"
	body := patchSessionRequest{
		MatchType: &mt, Surface: &sf, Position: &pos,
		Result: &res, Feeling: &feel, MatchTag: &tag,
	}

	rec := doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+created.ID, "Bearer "+token, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var updated session.Session
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.MatchType == nil || *updated.MatchType != "11-a-side" {
		t.Errorf("match_type not persisted: %+v", updated.MatchType)
	}
	if updated.Feeling == nil || *updated.Feeling != 4 {
		t.Errorf("feeling not persisted: %+v", updated.Feeling)
	}
	if updated.Result == nil || *updated.Result != "Won 3-1" {
		t.Errorf("result not persisted: %+v", updated.Result)
	}
	// Metrics unchanged.
	if updated.DistanceM != 8200 {
		t.Errorf("distance_m changed: %f", updated.DistanceM)
	}
}

func TestPatchSessionContext_StructuredResult_RoundTrip(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	var created session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	outcome := "win"
	score := "3-1"
	opp := "Los Tigres"
	comp := "league"
	goals := 2
	assists := 1
	notes := "Great second half"
	body := patchSessionRequest{
		Outcome: &outcome, Score: &score, Opponent: &opp,
		Competition: &comp, Goals: &goals, Assists: &assists, Notes: &notes,
	}

	rec := doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+created.ID, "Bearer "+token, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var updated session.Session
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Outcome == nil || *updated.Outcome != "win" {
		t.Errorf("outcome not persisted: %+v", updated.Outcome)
	}
	if updated.Score == nil || *updated.Score != "3-1" {
		t.Errorf("score not persisted: %+v", updated.Score)
	}
	if updated.Opponent == nil || *updated.Opponent != "Los Tigres" {
		t.Errorf("opponent not persisted: %+v", updated.Opponent)
	}
	if updated.Competition == nil || *updated.Competition != "league" {
		t.Errorf("competition not persisted: %+v", updated.Competition)
	}
	if updated.Goals == nil || *updated.Goals != 2 {
		t.Errorf("goals not persisted: %+v", updated.Goals)
	}
	if updated.Assists == nil || *updated.Assists != 1 {
		t.Errorf("assists not persisted: %+v", updated.Assists)
	}
	// Metrics unchanged.
	if updated.DistanceM != 8200 {
		t.Errorf("distance_m changed: %f", updated.DistanceM)
	}
}

func TestPatchSessionContext_InvalidOutcome_400(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	var created session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	bad := "tie"
	body := patchSessionRequest{Outcome: &bad}
	rec := doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+created.ID, "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPatchSessionContext_NegativeGoals_400(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	var created session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	g := -1
	body := patchSessionRequest{Goals: &g}
	rec := doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+created.ID, "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPatchSessionContext_InvalidEnum_400(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	var created session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	bad := "invalid-surface"
	body := patchSessionRequest{Surface: &bad}
	rec := doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+created.ID, "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPatchSessionContext_FeelingOutOfRange_400(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	var created session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	feel := 99
	body := patchSessionRequest{Feeling: &feel}
	rec := doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+created.ID, "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPatchSessionContext_NonOwner_404(t *testing.T) {
	store := newFakeSessionStore()
	tokenA := signToken(t, testSecret, "user-a", time.Now().Add(time.Hour))
	tokenB := signToken(t, testSecret, "user-b", time.Now().Add(time.Hour))
	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+tokenA, validSessionBody())
	var created session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	mt := "5-a-side"
	body := patchSessionRequest{MatchType: &mt}
	rec := doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+created.ID, "Bearer "+tokenB, body)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPatchSessionContext_PartialUpdate(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	var created session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	// Only set one field; others stay nil.
	res := "2-2 draw"
	body := patchSessionRequest{Result: &res}
	rec := doRequest(t, sessionDeps(store), http.MethodPatch, "/v1/sessions/"+created.ID, "Bearer "+token, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var updated session.Session
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Result == nil || *updated.Result != "2-2 draw" {
		t.Errorf("result not persisted: %+v", updated.Result)
	}
	if updated.MatchType != nil {
		t.Errorf("match_type unexpectedly set: %+v", updated.MatchType)
	}
}

func TestCreateSession_MatchRating_NoHR_NoSamples(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	rec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, validSessionBody())
	var created session.Session
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.MatchRating != nil {
		t.Errorf("match_rating should be nil without HR samples, got %v", *created.MatchRating)
	}
}

func TestCreateSession_Path_RoundTrip(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	body := validSessionBody()
	body.Path = []pathPointRequest{
		{TOffsetS: 0, Latitude: 14.0818, Longitude: -87.2068},
		{TOffsetS: 10, Latitude: 14.0820, Longitude: -87.2070},
		{TOffsetS: 20, Latitude: 14.0822, Longitude: -87.2072},
	}

	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body = %s", createRec.Code, createRec.Body.String())
	}
	var created session.Session
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	getRec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/"+created.ID, "Bearer "+token, nil)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getRec.Code)
	}
	var detail session.Session
	if err := json.NewDecoder(getRec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if len(detail.Path) != 3 {
		t.Fatalf("path length = %d, want 3", len(detail.Path))
	}
	if detail.Path[1].Latitude != 14.0820 || detail.Path[1].Longitude != -87.2070 {
		t.Errorf("path[1] = %+v, want lat 14.0820 lon -87.2070", detail.Path[1])
	}
}

func TestCreateSession_Path_InvalidLatitude_400(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	body := validSessionBody()
	body.Path = []pathPointRequest{{TOffsetS: 0, Latitude: 200, Longitude: 0}}

	rec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func createForRecords(t *testing.T, store *fakeSessionStore, token string, distance float64, sprints, durationS int) session.Session {
	t.Helper()
	body := validSessionBody()
	body.DistanceM = distance
	body.Sprints = sprints
	body.DurationS = durationS
	body.EndedAt = body.StartedAt.Add(time.Duration(durationS) * time.Second)
	rec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var created session.Session
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return created
}

func TestCreateSession_FirstSession_NoNewRecords(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	created := createForRecords(t, store, token, 5000, 5, 3000)
	if len(created.NewRecords) != 0 {
		t.Fatalf("first session should not break records, got %v", created.NewRecords)
	}
}

func TestCreateSession_BeatsDistance_FlagsNewRecord(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	_ = createForRecords(t, store, token, 5000, 5, 3000)
	beater := createForRecords(t, store, token, 9000, 20, 5400)
	if !contains(beater.NewRecords, "distance_m") {
		t.Fatalf("expected distance_m in new_records, got %v", beater.NewRecords)
	}
	if !contains(beater.NewRecords, "sprints") {
		t.Fatalf("expected sprints in new_records, got %v", beater.NewRecords)
	}
}

func TestCreateSession_DoesNotBeat_NoNewRecords(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	_ = createForRecords(t, store, token, 9000, 20, 5400)
	weak := createForRecords(t, store, token, 3000, 1, 2000)
	if len(weak.NewRecords) != 0 {
		t.Fatalf("weaker session should break no records, got %v", weak.NewRecords)
	}
}

func TestGetSession_FatigueDrop_Structured(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	hr := 170 // above 85% of 190 -> triggers high intensity
	hrMax := 190
	half1, half2 := 1, 2
	offset := 2700
	body := validSessionBody()
	body.Mode = session.ModeStructured
	body.HalftimeOffsetS = &offset
	body.HRMax = &hrMax
	for i := range 20 {
		h := half1
		if i >= 10 {
			h = half2
		}
		body.Samples = append(body.Samples, sampleRequest{
			TOffsetS: i * 300,
			HR:       &hr,
			Half:     &h,
		})
	}

	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body = %s", createRec.Code, createRec.Body.String())
	}
	var created session.Session
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	getRec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/"+created.ID, "Bearer "+token, nil)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getRec.Code)
	}
	var detail session.Session
	if err := json.NewDecoder(getRec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.FatigueDrop == nil {
		t.Fatal("fatigue_drop should not be nil for a structured session with valid halves")
	}
	if detail.FatigueDrop.FirstHalf.SampleCount < 6 {
		t.Errorf("first half sample count = %d, want ≥ 6", detail.FatigueDrop.FirstHalf.SampleCount)
	}
	if detail.FatigueDrop.SecondHalf.SampleCount < 6 {
		t.Errorf("second half sample count = %d, want ≥ 6", detail.FatigueDrop.SecondHalf.SampleCount)
	}
	if detail.FatigueDrop.HRAvgPctChange == nil {
		t.Error("hr_avg_pct_change should not be nil")
	}
	if detail.FatigueDrop.HighIntensityPctChange == nil {
		t.Error("high_intensity_pct_change should not be nil")
	}
}

func TestGetSession_FatigueDrop_NotEnoughSamples(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	hr := 150
	half1 := 1
	offset := 2700
	body := validSessionBody()
	body.Mode = session.ModeStructured
	body.HalftimeOffsetS = &offset
	body.Samples = []sampleRequest{
		{TOffsetS: 0, HR: &hr, Half: &half1},
		{TOffsetS: 10, HR: &hr, Half: &half1},
	}

	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	var created session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	getRec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/"+created.ID, "Bearer "+token, nil)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getRec.Code)
	}
	var detail session.Session
	if err := json.NewDecoder(getRec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.FatigueDrop != nil {
		t.Error("fatigue_drop should be nil for a session with insufficient samples")
	}
}

func TestGetSession_FatigueDrop_QuickMode(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	body := validSessionBody() // mode defaults to quick

	createRec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	var created session.Session
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	getRec := doRequest(t, sessionDeps(store), http.MethodGet, "/v1/sessions/"+created.ID, "Bearer "+token, nil)
	var detail session.Session
	if err := json.NewDecoder(getRec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.FatigueDrop != nil {
		t.Error("fatigue_drop should be nil for a quick session without halves")
	}
}

func TestCreateSession_MatchRating_Computed(t *testing.T) {
	store := newFakeSessionStore()
	token := signToken(t, testSecret, "user-1", time.Now().Add(time.Hour))
	hr := 150
	hrMax := 190
	body := validSessionBody()
	body.HRMax = &hrMax
	body.Samples = []sampleRequest{
		{TOffsetS: 0, HR: &hr},
		{TOffsetS: 60, HR: &hr},
		{TOffsetS: 120, HR: &hr},
	}
	rec := doRequest(t, sessionDeps(store), http.MethodPost, "/v1/sessions", "Bearer "+token, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var created session.Session
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.MatchRating == nil {
		t.Fatal("match_rating should not be nil")
	}
	if *created.MatchRating < 0 || *created.MatchRating > 100 {
		t.Errorf("match_rating out of 0–100 range: %f", *created.MatchRating)
	}
}
