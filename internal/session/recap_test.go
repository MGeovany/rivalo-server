package session

import (
	"testing"
	"time"
)

var recapNow = time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC) // Wednesday

func recapDaysAgo(n int) time.Time { return recapNow.AddDate(0, 0, -n) }

func TestBuildWeeklyRecap_AggregatesAndDeltas(t *testing.T) {
	r := func(v float64) *float64 { return &v }
	sessions := []RecapSession{
		// Current week: 2 matches.
		{ID: "c1", StartedAt: recapDaysAgo(0), DistanceM: 8000, Sprints: 12, MatchRating: r(80)},
		{ID: "c2", StartedAt: recapDaysAgo(1), DistanceM: 6000, Sprints: 8, MatchRating: r(70)},
		// Previous week: 1 match.
		{ID: "p1", StartedAt: recapDaysAgo(8), DistanceM: 7000, Sprints: 10, MatchRating: r(60)},
	}
	recap := BuildWeeklyRecap(sessions, recapNow)

	if recap.Current.MatchCount != 2 {
		t.Fatalf("current match count = %d, want 2", recap.Current.MatchCount)
	}
	if recap.Current.TotalDistanceM != 14000 {
		t.Errorf("current distance = %f, want 14000", recap.Current.TotalDistanceM)
	}
	if recap.Current.AvgRating == nil || *recap.Current.AvgRating != 75 {
		t.Errorf("current avg rating = %v, want 75", recap.Current.AvgRating)
	}
	if recap.Current.BestSessionID == nil || *recap.Current.BestSessionID != "c1" {
		t.Errorf("best session = %v, want c1", recap.Current.BestSessionID)
	}
	// Distance delta: (14000 - 7000)/7000 = 100%.
	if recap.DistanceDeltaPct == nil || *recap.DistanceDeltaPct != 100 {
		t.Errorf("distance delta = %v, want 100", recap.DistanceDeltaPct)
	}
	// Rating delta: (75 - 60)/60 = 25%.
	if recap.RatingDeltaPct == nil || *recap.RatingDeltaPct != 25 {
		t.Errorf("rating delta = %v, want 25", recap.RatingDeltaPct)
	}
}

func TestBuildWeeklyRecap_EmptyWeek(t *testing.T) {
	recap := BuildWeeklyRecap(nil, recapNow)
	if recap.Current.MatchCount != 0 {
		t.Fatalf("expected 0 matches, got %d", recap.Current.MatchCount)
	}
	if recap.DistanceDeltaPct != nil || recap.RatingDeltaPct != nil {
		t.Error("deltas should be nil with no previous-week data")
	}
}

func TestBuildWeeklyRecap_NoPreviousWeek_NoDelta(t *testing.T) {
	sessions := []RecapSession{
		{ID: "c1", StartedAt: recapDaysAgo(0), DistanceM: 8000, Sprints: 12},
	}
	recap := BuildWeeklyRecap(sessions, recapNow)
	if recap.Current.MatchCount != 1 {
		t.Fatalf("current = %d, want 1", recap.Current.MatchCount)
	}
	if recap.DistanceDeltaPct != nil {
		t.Errorf("distance delta should be nil without previous week, got %v", *recap.DistanceDeltaPct)
	}
}
