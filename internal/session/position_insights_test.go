package session

import (
	"strings"
	"testing"
)

func TestAssemblePositionInsights_BelowThreshold_NotEnough(t *testing.T) {
	// Only one position clears MinSessionsPerPosition.
	all := []PositionStat{
		{Position: "midfielder", SessionCount: 4, AvgDistanceM: fp(8000)},
		{Position: "defender", SessionCount: 2, AvgDistanceM: fp(5000)},
	}
	got := AssemblePositionInsights(all)
	if got.HasEnoughData {
		t.Fatal("expected has_enough_data=false with only one qualifying position")
	}
	if len(got.Comparisons) != 0 {
		t.Fatalf("expected no comparisons, got %v", got.Comparisons)
	}
}

func TestAssemblePositionInsights_EnoughData_BuildsComparisons(t *testing.T) {
	all := []PositionStat{
		{Position: "midfielder", SessionCount: 5, AvgDistanceM: fp(9000), AvgSprints: fp(18), AvgIntensity: fp(75)},
		{Position: "defender", SessionCount: 4, AvgDistanceM: fp(6000), AvgSprints: fp(10), AvgIntensity: fp(60)},
	}
	got := AssemblePositionInsights(all)
	if !got.HasEnoughData {
		t.Fatal("expected has_enough_data=true")
	}
	if len(got.Positions) != 2 {
		t.Fatalf("expected 2 qualifying positions, got %d", len(got.Positions))
	}
	// Most-played first.
	if got.Positions[0].Position != "midfielder" {
		t.Errorf("expected midfielder first (most sessions), got %s", got.Positions[0].Position)
	}
	if len(got.Comparisons) == 0 {
		t.Fatal("expected physical comparisons")
	}
}

func TestBuildPositionComparisons_NeverDeclaresBestPosition(t *testing.T) {
	stats := []PositionStat{
		{Position: "winger", SessionCount: 5, AvgDistanceM: fp(9500), AvgSprints: fp(22), AvgIntensity: fp(80)},
		{Position: "defender", SessionCount: 5, AvgDistanceM: fp(6000), AvgSprints: fp(9), AvgIntensity: fp(58)},
	}
	for _, c := range BuildPositionComparisons(stats) {
		lc := strings.ToLower(c)
		if strings.Contains(lc, "best position") || strings.Contains(lc, "you should play") {
			t.Errorf("comparison must not declare a best position: %q", c)
		}
	}
}

func TestBuildPositionComparisons_SkipsTinyGaps(t *testing.T) {
	stats := []PositionStat{
		{Position: "a", SessionCount: 5, AvgDistanceM: fp(8000), AvgSprints: fp(12), AvgIntensity: fp(70)},
		{Position: "b", SessionCount: 5, AvgDistanceM: fp(8050), AvgSprints: fp(12), AvgIntensity: fp(71)},
	}
	if got := BuildPositionComparisons(stats); len(got) != 0 {
		t.Fatalf("expected no comparisons for negligible gaps, got %v", got)
	}
}
