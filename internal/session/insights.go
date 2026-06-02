package session

import "fmt"

// MinInsightSessions is the minimum number of sessions required before any
// rule-based insight is produced (V2-H). Below this, BuildInsights returns nil.
const MinInsightSessions = 5

// BuildInsights derives explainable, rule-based statements from aggregate
// session stats. It is pure and deterministic — every insight cites the numbers
// behind it, and none are produced below MinInsightSessions. `recentRatingAvg`
// is the average Match Rating of the user's most recent sessions (nil if
// unavailable), used for the recent-trend rule.
func BuildInsights(ins SessionInsights, recentRatingAvg *float64) []Insight {
	if ins.Totals.SessionCount < MinInsightSessions {
		return nil
	}

	var out []Insight

	// Rule 1: best context by Match Rating (where two+ contexts have ratings).
	if in := bestByRating(ins.BySurface, "surface", "surface"); in != nil {
		out = append(out, *in)
	}
	if in := bestByRating(ins.ByPosition, "position", "position"); in != nil {
		out = append(out, *in)
	}

	// Rule 2: most played context (position preferred, else match type).
	if in := mostPlayed(ins.ByPosition, "position"); in != nil {
		out = append(out, *in)
	} else if in := mostPlayed(ins.ByMatchType, "match type"); in != nil {
		out = append(out, *in)
	}

	// Rule 3: distance standout vs the overall per-match average.
	if ins.Averages.DistancePerMatch != nil {
		if in := distanceStandout(ins.ByMatchType, *ins.Averages.DistancePerMatch, "match_type"); in != nil {
			out = append(out, *in)
		}
	}

	// Rule 4: recent Match Rating trend vs the all-time average.
	if recentRatingAvg != nil && ins.Averages.MatchRating != nil {
		if in := recentTrend(*recentRatingAvg, *ins.Averages.MatchRating); in != nil {
			out = append(out, *in)
		}
	}

	return out
}

// bestByRating finds the highest- and lowest-rated context group and, when they
// differ meaningfully, returns an insight comparing them.
func bestByRating(groups []ContextGroup, kindSuffix, label string) *Insight {
	type rated struct {
		value  string
		rating float64
	}
	var rs []rated
	for _, g := range groups {
		if g.AvgMatchRating != nil && g.Count >= 2 {
			rs = append(rs, rated{g.Value, *g.AvgMatchRating})
		}
	}
	if len(rs) < 2 {
		return nil
	}
	best, worst := rs[0], rs[0]
	for _, r := range rs[1:] {
		if r.rating > best.rating {
			best = r
		}
		if r.rating < worst.rating {
			worst = r
		}
	}
	// Require a non-trivial gap to avoid noise.
	if best.rating-worst.rating < 5 {
		return nil
	}
	return &Insight{
		Kind:  "best_" + kindSuffix,
		Title: fmt.Sprintf("You perform best on %s", best.value),
		Detail: fmt.Sprintf("Avg rating %.0f on %s vs %.0f on %s.",
			best.rating, best.value, worst.rating, worst.value),
	}
}

// mostPlayed returns an insight for the dominant context value, when one clearly
// leads.
func mostPlayed(groups []ContextGroup, label string) *Insight {
	if len(groups) < 2 {
		return nil
	}
	top := groups[0]
	total := 0
	for _, g := range groups {
		total += g.Count
		if g.Count > top.Count {
			top = g
		}
	}
	if total == 0 || top.Count*2 <= total {
		// No clear majority (top is not more than half of all sessions).
		return nil
	}
	return &Insight{
		Kind:   "most_played",
		Title:  fmt.Sprintf("You play most as %s", top.value(label)),
		Detail: fmt.Sprintf("%d of %d sessions.", top.Count, total),
	}
}

// distanceStandout flags the context where the player covers clearly more ground
// than their overall average.
func distanceStandout(groups []ContextGroup, overallAvg float64, kindSuffix string) *Insight {
	if overallAvg <= 0 {
		return nil
	}
	var top *ContextGroup
	for i := range groups {
		g := groups[i]
		if g.AvgDistance == nil || g.Count < 2 {
			continue
		}
		if top == nil || *g.AvgDistance > *top.AvgDistance {
			top = &groups[i]
		}
	}
	if top == nil || top.AvgDistance == nil {
		return nil
	}
	// Require at least 15% above the overall average.
	if *top.AvgDistance < overallAvg*1.15 {
		return nil
	}
	return &Insight{
		Kind:  "distance_standout_" + kindSuffix,
		Title: fmt.Sprintf("You run more in %s", top.Value),
		Detail: fmt.Sprintf("%.1f km vs %.1f km average.",
			*top.AvgDistance/1000, overallAvg/1000),
	}
}

// recentTrend compares recent vs all-time average Match Rating.
func recentTrend(recent, allTime float64) *Insight {
	if allTime <= 0 {
		return nil
	}
	change := (recent - allTime) / allTime * 100
	if change >= 5 {
		return &Insight{
			Kind:   "recent_trend_up",
			Title:  "Your form is trending up",
			Detail: fmt.Sprintf("Recent rating %.0f vs %.0f average (+%.0f%%).", recent, allTime, change),
		}
	}
	if change <= -5 {
		return &Insight{
			Kind:   "recent_trend_down",
			Title:  "Your recent form has dipped",
			Detail: fmt.Sprintf("Recent rating %.0f vs %.0f average (%.0f%%).", recent, allTime, change),
		}
	}
	return nil
}

// value returns the group value with a sensible fallback label.
func (g ContextGroup) value(label string) string {
	if g.Value == "" {
		return label
	}
	return g.Value
}
