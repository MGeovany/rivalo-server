package session

import (
	"testing"
	"time"
)

// anchor "now" — a Wednesday, so the current ISO week is well-defined.
var streakNow = time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)

func daysAgo(n int) time.Time { return streakNow.AddDate(0, 0, -n) }

func TestBuildStreaks_ConsecutiveWeeks(t *testing.T) {
	// One match this week and one in each of the previous 3 weeks → 4-week streak.
	sessions := []StreakSession{
		{StartedAt: daysAgo(0)},
		{StartedAt: daysAgo(7)},
		{StartedAt: daysAgo(14)},
		{StartedAt: daysAgo(21)},
	}
	s := BuildStreaks(sessions, streakNow)
	if s.CurrentWeeks != 4 {
		t.Fatalf("current weeks = %d, want 4", s.CurrentWeeks)
	}
	if s.BestWeeks != 4 {
		t.Fatalf("best weeks = %d, want 4", s.BestWeeks)
	}
}

func TestBuildStreaks_GapBreaksStreak(t *testing.T) {
	// Played this week, then a gap (nothing 1 & 2 weeks ago), then 3 weeks ago.
	sessions := []StreakSession{
		{StartedAt: daysAgo(0)},
		{StartedAt: daysAgo(21)},
	}
	s := BuildStreaks(sessions, streakNow)
	if s.CurrentWeeks != 1 {
		t.Fatalf("current weeks = %d, want 1 (gap should break)", s.CurrentWeeks)
	}
}

func TestBuildStreaks_CurrentWeekEmptyButLastWeekCounts(t *testing.T) {
	// Nothing this week yet, but last week and the week before → streak alive (2).
	sessions := []StreakSession{
		{StartedAt: daysAgo(7)},
		{StartedAt: daysAgo(14)},
	}
	s := BuildStreaks(sessions, streakNow)
	if s.CurrentWeeks != 2 {
		t.Fatalf("current weeks = %d, want 2", s.CurrentWeeks)
	}
}

func TestBuildStreaks_StaleBreaksCurrent(t *testing.T) {
	// Last match 3 weeks ago → a closed empty week exists → current streak 0.
	sessions := []StreakSession{{StartedAt: daysAgo(21)}}
	s := BuildStreaks(sessions, streakNow)
	if s.CurrentWeeks != 0 {
		t.Fatalf("current weeks = %d, want 0", s.CurrentWeeks)
	}
	if s.BestWeeks != 1 {
		t.Fatalf("best weeks = %d, want 1", s.BestWeeks)
	}
}

func special(s Streaks, kind string) SpecialStreak {
	for _, sp := range s.Special {
		if sp.Kind == kind {
			return sp
		}
	}
	return SpecialStreak{}
}

func TestBuildStreaks_SprintStreakActive(t *testing.T) {
	sessions := []StreakSession{
		{StartedAt: daysAgo(0), Sprints: 22},
		{StartedAt: daysAgo(3), Sprints: 25},
		{StartedAt: daysAgo(6), Sprints: 20},
		{StartedAt: daysAgo(9), Sprints: 8}, // breaks
	}
	sp := special(BuildStreaks(sessions, streakNow), "sprints")
	if sp.Count != 3 || !sp.Active {
		t.Fatalf("sprint streak = %+v, want count 3 active", sp)
	}
}

func TestBuildStreaks_RatingImprovingActive(t *testing.T) {
	r := func(v float64) *float64 { return &v }
	// Most recent first: 90 > 86 > 82 > 70 → improving over time, run of 4.
	sessions := []StreakSession{
		{StartedAt: daysAgo(0), MatchRating: r(90)},
		{StartedAt: daysAgo(3), MatchRating: r(86)},
		{StartedAt: daysAgo(6), MatchRating: r(82)},
		{StartedAt: daysAgo(9), MatchRating: r(70)},
	}
	sp := special(BuildStreaks(sessions, streakNow), "rating_improving")
	if sp.Count != 4 || !sp.Active {
		t.Fatalf("rating streak = %+v, want count 4 active", sp)
	}
}

func TestBuildStreaks_FatigueControlledActive(t *testing.T) {
	yes := true
	no := false
	sessions := []StreakSession{
		{StartedAt: daysAgo(0), Structured: true, FatigueControlled: &yes},
		{StartedAt: daysAgo(3), Structured: true, FatigueControlled: &yes},
		{StartedAt: daysAgo(6), Structured: true, FatigueControlled: &yes},
		{StartedAt: daysAgo(9), Structured: true, FatigueControlled: &no}, // breaks
	}
	sp := special(BuildStreaks(sessions, streakNow), "fatigue_controlled")
	if sp.Count != 3 || !sp.Active {
		t.Fatalf("fatigue streak = %+v, want count 3 active", sp)
	}
}
