package session

import "testing"

func fp(v float64) *float64 { return &v }

func hasKind(insights []Insight, kind string) bool {
	for _, in := range insights {
		if in.Kind == kind {
			return true
		}
	}
	return false
}

func TestBuildInsights_BelowThreshold_Empty(t *testing.T) {
	ins := SessionInsights{
		Totals:   StatsTotals{SessionCount: MinInsightSessions - 1},
		Averages: StatsAverages{MatchRating: fp(70)},
		BySurface: []ContextGroup{
			{Value: "grass", Count: 3, AvgMatchRating: fp(80)},
			{Value: "turf", Count: 3, AvgMatchRating: fp(60)},
		},
	}
	if got := BuildInsights(ins, fp(90)); got != nil {
		t.Fatalf("expected nil insights below threshold, got %v", got)
	}
}

func TestBuildInsights_BestSurfaceByRating(t *testing.T) {
	ins := SessionInsights{
		Totals:   StatsTotals{SessionCount: 8},
		Averages: StatsAverages{DistancePerMatch: fp(6000), MatchRating: fp(70)},
		BySurface: []ContextGroup{
			{Value: "grass", Count: 4, AvgMatchRating: fp(82)},
			{Value: "turf", Count: 4, AvgMatchRating: fp(61)},
		},
	}
	got := BuildInsights(ins, nil)
	if !hasKind(got, "best_surface") {
		t.Fatalf("expected a best_surface insight, got %+v", got)
	}
}

func TestBuildInsights_BestSurface_SkippedWhenGapTooSmall(t *testing.T) {
	ins := SessionInsights{
		Totals: StatsTotals{SessionCount: 8},
		BySurface: []ContextGroup{
			{Value: "grass", Count: 4, AvgMatchRating: fp(72)},
			{Value: "turf", Count: 4, AvgMatchRating: fp(70)},
		},
	}
	if hasKind(BuildInsights(ins, nil), "best_surface") {
		t.Fatal("did not expect best_surface insight for a <5 point gap")
	}
}

func TestBuildInsights_MostPlayedPosition(t *testing.T) {
	ins := SessionInsights{
		Totals: StatsTotals{SessionCount: 10},
		ByPosition: []ContextGroup{
			{Value: "midfielder", Count: 7},
			{Value: "defender", Count: 3},
		},
	}
	if !hasKind(BuildInsights(ins, nil), "most_played") {
		t.Fatalf("expected a most_played insight")
	}
}

func TestBuildInsights_RecentTrendUp(t *testing.T) {
	ins := SessionInsights{
		Totals:   StatsTotals{SessionCount: 6},
		Averages: StatsAverages{MatchRating: fp(60)},
	}
	got := BuildInsights(ins, fp(72)) // +20% vs all-time
	if !hasKind(got, "recent_trend_up") {
		t.Fatalf("expected recent_trend_up, got %+v", got)
	}
}

func TestBuildInsights_DistanceStandout(t *testing.T) {
	ins := SessionInsights{
		Totals:   StatsTotals{SessionCount: 7},
		Averages: StatsAverages{DistancePerMatch: fp(5000)},
		ByMatchType: []ContextGroup{
			{Value: "11-a-side", Count: 3, AvgDistance: fp(8000)},
			{Value: "5-a-side", Count: 4, AvgDistance: fp(3500)},
		},
	}
	if !hasKind(BuildInsights(ins, nil), "distance_standout_match_type") {
		t.Fatalf("expected a distance_standout insight")
	}
}
