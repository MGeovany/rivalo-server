package session

import "fmt"

// MinMatchInsightPrior is the minimum number of prior sessions required before
// BuildMatchInsights produces any insights.
const MinMatchInsightPrior = 2

// BuildMatchInsights compares a just-completed session against the user's prior
// history to produce 1-3 short explainable observations. It is pure and
// deterministic — results are nil when there are fewer than MinMatchInsightPrior
// prior sessions or no metric exceeds its threshold.
func BuildMatchInsights(current Session, prior []Session) []MatchInsight {
	if len(prior) < MinMatchInsightPrior {
		return nil
	}

	var out []MatchInsight

	avgDistance := avgFloat(prior, func(s Session) float64 { return s.DistanceM })
	if current.DistanceM > avgDistance*1.2 {
		out = append(out, MatchInsight{
			Kind:    "distance_burst",
			Title:   "Longest distance",
			Message: fmt.Sprintf("%.1f km — your best in %d matches", current.DistanceM/1000, len(prior)+1),
		})
	}

	avgSprints := avgFloat(prior, func(s Session) float64 { return float64(s.Sprints) })
	if avgSprints > 0 && float64(current.Sprints) > avgSprints*1.3 {
		out = append(out, MatchInsight{
			Kind:    "sprint_peak",
			Title:   "Most sprints",
			Message: fmt.Sprintf("%d sprints — your highest in %d matches", current.Sprints, len(prior)+1),
		})
	}

	avgDuration := avgFloat(prior, func(s Session) float64 { return float64(s.DurationS) })
	if current.DurationS > int(avgDuration*1.25) {
		out = append(out, MatchInsight{
			Kind:    "duration_record",
			Title:   "Longest match",
			Message: fmt.Sprintf("%d min — your longest in %d matches", current.DurationS/60, len(prior)+1),
		})
	}

	if current.MatchRating != nil {
		avgRating := avgFloatPtr(prior, func(s Session) *float64 { return s.MatchRating })
		if avgRating != nil && *current.MatchRating > *avgRating*1.1 {
			out = append(out, MatchInsight{
				Kind:    "rating_boost",
				Title:   "Best rating",
				Message: fmt.Sprintf("%.0f/100 — your best match rating in %d matches", *current.MatchRating, len(prior)+1),
			})
		}
	}

	if current.Intensity != nil {
		avgIntensity := avgFloatPtr(prior, func(s Session) *float64 { return s.Intensity })
		if avgIntensity != nil && *current.Intensity > *avgIntensity*1.15 {
			out = append(out, MatchInsight{
				Kind:    "intensity_peak",
				Title:   "Highest intensity",
				Message: fmt.Sprintf("%.0f%% — your most intense match in %d matches", *current.Intensity*100, len(prior)+1),
			})
		}
	}

	if len(out) == 0 {
		return nil
	}
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

// avgFloat returns the average of a float64 field across sessions.
func avgFloat[S any](sessions []S, f func(S) float64) float64 {
	if len(sessions) == 0 {
		return 0
	}
	var sum float64
	for _, s := range sessions {
		sum += f(s)
	}
	return sum / float64(len(sessions))
}

// avgFloatPtr returns the average of an optional float64 field across sessions.
// Returns nil when no session has a non-nil value.
func avgFloatPtr[S any](sessions []S, f func(S) *float64) *float64 {
	var sum float64
	var n int
	for _, s := range sessions {
		if v := f(s); v != nil {
			sum += *v
			n++
		}
	}
	if n == 0 {
		return nil
	}
	v := sum / float64(n)
	return &v
}
