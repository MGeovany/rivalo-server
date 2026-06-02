package session

import (
	"testing"
	"time"
)

func TestBuildRivalries_AllOutcomes(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	outcome := func(s string) *string { return &s }

	sessions := []RivalSession{
		{Opponent: "FC Barcelona", Outcome: outcome(OutcomeWin), StartedAt: now.AddDate(0, 0, -2), DistanceM: 8000, Sprints: 15, MatchRating: ptr(85.0)},
		{Opponent: "FC Barcelona", Outcome: outcome(OutcomeDraw), StartedAt: now.AddDate(0, 0, -9), DistanceM: 7500, Sprints: 12, MatchRating: ptr(78.0)},
		{Opponent: "FC Barcelona", Outcome: outcome(OutcomeLoss), StartedAt: now.AddDate(0, 0, -16), DistanceM: 8200, Sprints: 10},
		{Opponent: "Real Madrid", Outcome: outcome(OutcomeWin), StartedAt: now.AddDate(0, 0, -3), DistanceM: 9000, Sprints: 18, MatchRating: ptr(90.0)},
		{Opponent: "Real Madrid", Outcome: outcome(OutcomeLoss), StartedAt: now.AddDate(0, 0, -17), DistanceM: 7800, Sprints: 14, MatchRating: ptr(82.0)},
	}

	rivals := BuildRivalries(sessions)
	if len(rivals) != 2 {
		t.Fatalf("got %d rivalries, want 2", len(rivals))
	}

	barca := rivals[0]
	if barca.Opponent != "FC Barcelona" {
		t.Fatalf("opponent = %q, want FC Barcelona", barca.Opponent)
	}
	if barca.MatchCount != 3 {
		t.Fatalf("match_count = %d, want 3", barca.MatchCount)
	}
	if barca.Wins != 1 || barca.Draws != 1 || barca.Losses != 1 {
		t.Fatalf("W/D/L = %d/%d/%d, want 1/1/1", barca.Wins, barca.Draws, barca.Losses)
	}
	if barca.AvgRating == nil || *barca.AvgRating != 81.5 {
		t.Fatalf("avg_rating = %v, want 81.5", barca.AvgRating)
	}
}

func TestBuildRivalries_BelowThreshold(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	sessions := []RivalSession{
		{Opponent: "One-off FC", Outcome: ptr(OutcomeWin), StartedAt: now.AddDate(0, 0, -1), DistanceM: 5000, Sprints: 5},
	}
	rivals := BuildRivalries(sessions)
	if len(rivals) != 0 {
		t.Fatalf("got %d rivalries, want 0 (below threshold of %d)", len(rivals), MinRivalryMatches)
	}
}

func TestBuildRivalries_NoOutcomeCounts(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	sessions := []RivalSession{
		{Opponent: "Mystery Team", StartedAt: now.AddDate(0, 0, -4), DistanceM: 6000, Sprints: 8},
		{Opponent: "Mystery Team", StartedAt: now.AddDate(0, 0, -11), DistanceM: 5500, Sprints: 6},
	}
	rivals := BuildRivalries(sessions)
	if len(rivals) != 1 {
		t.Fatalf("got %d rivalries, want 1", len(rivals))
	}
	if rivals[0].Wins != 0 || rivals[0].Draws != 0 || rivals[0].Losses != 0 {
		t.Fatalf("W/D/L = %d/%d/%d, want 0/0/0 when no outcomes set", rivals[0].Wins, rivals[0].Draws, rivals[0].Losses)
	}
}

func TestBuildRivalries_EmptySessions(t *testing.T) {
	rivals := BuildRivalries(nil)
	if len(rivals) != 0 {
		t.Fatalf("got %d rivalries, want 0", len(rivals))
	}
}
