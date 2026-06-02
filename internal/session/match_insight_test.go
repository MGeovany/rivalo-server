package session

import "testing"

func TestBuildMatchInsights_BelowThreshold(t *testing.T) {
	current := Session{DistanceM: 8000, Sprints: 30, DurationS: 5400}
	prior := []Session{{DistanceM: 5000, Sprints: 15, DurationS: 3600}}
	if got := BuildMatchInsights(current, prior); got != nil {
		t.Fatalf("expected nil below threshold, got %v", got)
	}
}

func TestBuildMatchInsights_DistanceBurst(t *testing.T) {
	current := Session{DistanceM: 9500, Sprints: 15, DurationS: 4200}
	prior := []Session{
		{DistanceM: 5000, Sprints: 12, DurationS: 3600},
		{DistanceM: 6000, Sprints: 18, DurationS: 3900},
		{DistanceM: 5500, Sprints: 14, DurationS: 3700},
	}
	got := BuildMatchInsights(current, prior)
	if !hasMatchKind(got, "distance_burst") {
		t.Fatalf("expected distance_burst insight, got %+v", got)
	}
}

func TestBuildMatchInsights_SprintPeak(t *testing.T) {
	current := Session{DistanceM: 5000, Sprints: 30, DurationS: 3600}
	prior := []Session{
		{DistanceM: 4800, Sprints: 10, DurationS: 3500},
		{DistanceM: 5200, Sprints: 12, DurationS: 3700},
	}
	got := BuildMatchInsights(current, prior)
	if !hasMatchKind(got, "sprint_peak") {
		t.Fatalf("expected sprint_peak insight, got %+v", got)
	}
}

func TestBuildMatchInsights_RatingBoost(t *testing.T) {
	current := Session{DistanceM: 5000, Sprints: 15, DurationS: 3600, MatchRating: fp(85)}
	prior := []Session{
		{DistanceM: 4800, Sprints: 12, DurationS: 3500, MatchRating: fp(65)},
		{DistanceM: 5200, Sprints: 14, DurationS: 3700, MatchRating: fp(70)},
	}
	got := BuildMatchInsights(current, prior)
	if !hasMatchKind(got, "rating_boost") {
		t.Fatalf("expected rating_boost insight, got %+v", got)
	}
}

func TestBuildMatchInsights_IntensityPeak(t *testing.T) {
	current := Session{DistanceM: 5000, Sprints: 15, DurationS: 3600, Intensity: fp(0.85)}
	prior := []Session{
		{DistanceM: 4800, Sprints: 12, DurationS: 3500, Intensity: fp(0.60)},
		{DistanceM: 5200, Sprints: 14, DurationS: 3700, Intensity: fp(0.65)},
	}
	got := BuildMatchInsights(current, prior)
	if !hasMatchKind(got, "intensity_peak") {
		t.Fatalf("expected intensity_peak insight, got %+v", got)
	}
}

func TestBuildMatchInsights_NilWhenNoRecord(t *testing.T) {
	current := Session{DistanceM: 5000, Sprints: 12, DurationS: 3600}
	prior := []Session{
		{DistanceM: 4800, Sprints: 10, DurationS: 3500},
		{DistanceM: 5200, Sprints: 14, DurationS: 3700},
	}
	got := BuildMatchInsights(current, prior)
	if got != nil {
		t.Fatalf("expected nil when no metric stands out, got %+v", got)
	}
}

func TestBuildMatchInsights_LimitToThree(t *testing.T) {
	current := Session{DistanceM: 9500, Sprints: 35, DurationS: 5400, MatchRating: fp(90), Intensity: fp(0.85)}
	prior := []Session{
		{DistanceM: 5000, Sprints: 10, DurationS: 3000, MatchRating: fp(60), Intensity: fp(0.50)},
		{DistanceM: 6000, Sprints: 12, DurationS: 3200, MatchRating: fp(65), Intensity: fp(0.55)},
	}
	got := BuildMatchInsights(current, prior)
	if len(got) > 3 {
		t.Fatalf("expected at most 3 insights, got %d", len(got))
	}
	if got == nil {
		t.Fatal("expected insights, got nil")
	}
}

func hasMatchKind(insights []MatchInsight, kind string) bool {
	for _, in := range insights {
		if in.Kind == kind {
			return true
		}
	}
	return false
}
