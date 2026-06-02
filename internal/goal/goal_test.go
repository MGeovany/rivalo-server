package goal

import (
	"testing"
	"time"
)

var goalNow = time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC) // Wednesday

func daysAgo(n int) time.Time { return goalNow.AddDate(0, 0, -n) }

func TestCalculateProgress_WeeklyDistance(t *testing.T) {
	g := Goal{Metric: "distance", Period: "week", Target: 15000}
	sessions := []GoalSession{
		{StartedAt: daysAgo(0), DistanceM: 8000},
		{StartedAt: daysAgo(1), DistanceM: 6000},
		{StartedAt: daysAgo(8), DistanceM: 5000}, // last week, excluded
	}
	p, na := CalculateProgress(g, sessions, goalNow)
	if p != 14000 {
		t.Fatalf("progress = %.0f, want 14000", p)
	}
	if na {
		t.Fatal("should not be newly achieved (14000 < 15000)")
	}
}

func TestCalculateProgress_NewlyAchieved(t *testing.T) {
	// Target already met this week; not yet achieved.
	g := Goal{Metric: "distance", Period: "week", Target: 10000}
	sessions := []GoalSession{
		{StartedAt: daysAgo(0), DistanceM: 12000},
	}
	p, na := CalculateProgress(g, sessions, goalNow)
	if p != 12000 {
		t.Fatalf("progress = %.0f, want 12000", p)
	}
	if !na {
		t.Fatal("should be newly achieved")
	}
}

func TestCalculateProgress_AlreadyAchieved(t *testing.T) {
	at := goalNow.AddDate(0, 0, -1)
	g := Goal{Metric: "matches", Period: "week", Target: 3, AchievedAt: &at}
	sessions := []GoalSession{
		{StartedAt: daysAgo(0)},
		{StartedAt: daysAgo(1)},
		{StartedAt: daysAgo(2)},
	}
	p, na := CalculateProgress(g, sessions, goalNow)
	if p != 3 {
		t.Fatalf("progress = %.0f, want 3", p)
	}
	if na {
		t.Fatal("already achieved, should not be newly achieved")
	}
}

func TestCalculateProgress_WeeklyMatches(t *testing.T) {
	g := Goal{Metric: "matches", Period: "week", Target: 5}
	sessions := []GoalSession{
		{StartedAt: daysAgo(0)},
		{StartedAt: daysAgo(1)},
		{StartedAt: daysAgo(2)},
		{StartedAt: daysAgo(8)}, // last week
	}
	p, _ := CalculateProgress(g, sessions, goalNow)
	if p != 3 {
		t.Fatalf("progress = %.0f, want 3", p)
	}
}

func TestCalculateProgress_WeeklySprints(t *testing.T) {
	g := Goal{Metric: "sprints", Period: "week", Target: 50}
	sessions := []GoalSession{
		{StartedAt: daysAgo(0), Sprints: 22},
		{StartedAt: daysAgo(1), Sprints: 18},
	}
	p, _ := CalculateProgress(g, sessions, goalNow)
	if p != 40 {
		t.Fatalf("progress = %.0f, want 40", p)
	}
}

func TestCalculateProgress_WeeklyRating(t *testing.T) {
	r := func(v float64) *float64 { return &v }
	g := Goal{Metric: "rating", Period: "week", Target: 80}
	sessions := []GoalSession{
		{StartedAt: daysAgo(0), MatchRating: r(82)},
		{StartedAt: daysAgo(1), MatchRating: r(78)},
		{StartedAt: daysAgo(8), MatchRating: r(90)}, // last week
	}
	p, _ := CalculateProgress(g, sessions, goalNow)
	if p != 80 {
		t.Fatalf("progress = %.0f, want 80", p)
	}
}

func TestCalculateProgress_EmptySessions(t *testing.T) {
	g := Goal{Metric: "matches", Period: "week", Target: 1}
	p, na := CalculateProgress(g, nil, goalNow)
	if p != 0 {
		t.Fatalf("progress = %.0f, want 0", p)
	}
	if na {
		t.Fatal("no sessions, should not be newly achieved")
	}
}

func TestCalculateProgress_NoRatingSessions(t *testing.T) {
	g := Goal{Metric: "rating", Period: "week", Target: 80}
	sessions := []GoalSession{
		{StartedAt: daysAgo(0), DistanceM: 5000},
	}
	p, _ := CalculateProgress(g, sessions, goalNow)
	if p != 0 {
		t.Fatalf("progress = %.0f, want 0 (no ratings)", p)
	}
}
