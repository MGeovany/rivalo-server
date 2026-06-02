package session

import (
	"fmt"
	"sort"
)

const (
	// MinSessionsPerPosition is the minimum sessions a position needs before it
	// is considered for comparison.
	MinSessionsPerPosition = 3
	// MinPositionsToCompare is the minimum number of qualifying positions needed
	// to draw any comparison.
	MinPositionsToCompare = 2
)

// PositionStat holds the physical averages for one playing position.
type PositionStat struct {
	Position       string   `json:"position"`
	SessionCount   int      `json:"session_count"`
	AvgDistanceM   *float64 `json:"avg_distance_m"`
	AvgSprints     *float64 `json:"avg_sprints"`
	AvgIntensity   *float64 `json:"avg_intensity"`
	AvgMatchRating *float64 `json:"avg_match_rating"`
	AvgDurationS   *float64 `json:"avg_duration_s"`
}

// PositionInsights is a deliberately cautious, physical-only comparison across
// positions. It never declares a "best" position — only physical tendencies.
type PositionInsights struct {
	HasEnoughData bool           `json:"has_enough_data"`
	Positions     []PositionStat `json:"positions"`
	Comparisons   []string       `json:"comparisons"`
}

// AssemblePositionInsights filters the per-position stats to those with enough
// sessions and, when at least two qualify, builds neutral physical comparisons.
// `all` may contain every position the user has played; only qualifying ones are
// surfaced.
func AssemblePositionInsights(all []PositionStat) PositionInsights {
	qualifying := make([]PositionStat, 0, len(all))
	for _, p := range all {
		if p.SessionCount >= MinSessionsPerPosition {
			qualifying = append(qualifying, p)
		}
	}
	sort.SliceStable(qualifying, func(i, j int) bool {
		if qualifying[i].SessionCount != qualifying[j].SessionCount {
			return qualifying[i].SessionCount > qualifying[j].SessionCount
		}
		return qualifying[i].Position < qualifying[j].Position
	})

	if len(qualifying) < MinPositionsToCompare {
		return PositionInsights{HasEnoughData: false, Positions: qualifying}
	}
	return PositionInsights{
		HasEnoughData: true,
		Positions:     qualifying,
		Comparisons:   BuildPositionComparisons(qualifying),
	}
}

// BuildPositionComparisons produces neutral, explainable statements about
// physical tendencies between positions. It deliberately avoids any "best
// position" verdict — every statement is about measured physical load.
func BuildPositionComparisons(stats []PositionStat) []string {
	if len(stats) < MinPositionsToCompare {
		return nil
	}
	var out []string

	if hi, lo, ok := extremes(stats, func(p PositionStat) *float64 { return p.AvgDistanceM }); ok {
		if *hi.val > *lo.val*1.10 {
			out = append(out, fmt.Sprintf(
				"You cover more ground as %s (%.1f km) than as %s (%.1f km).",
				hi.pos, *hi.val/1000, lo.pos, *lo.val/1000))
		}
	}
	if hi, lo, ok := extremes(stats, func(p PositionStat) *float64 { return p.AvgSprints }); ok {
		if *hi.val > *lo.val+1 {
			out = append(out, fmt.Sprintf(
				"You sprint more as %s (%.0f vs %.0f per match).",
				hi.pos, *hi.val, *lo.val))
		}
	}
	if hi, lo, ok := extremes(stats, func(p PositionStat) *float64 { return p.AvgIntensity }); ok {
		if *hi.val > *lo.val+5 {
			out = append(out, fmt.Sprintf(
				"Your physical intensity runs higher as %s (%.0f vs %.0f).",
				hi.pos, *hi.val, *lo.val))
		}
	}
	return out
}

type posVal struct {
	pos string
	val *float64
}

// extremes returns the highest and lowest position by the selected metric,
// considering only positions where the metric is present.
func extremes(stats []PositionStat, pick func(PositionStat) *float64) (hi, lo posVal, ok bool) {
	var vals []posVal
	for _, p := range stats {
		if v := pick(p); v != nil {
			vals = append(vals, posVal{p.Position, v})
		}
	}
	if len(vals) < 2 {
		return posVal{}, posVal{}, false
	}
	hi, lo = vals[0], vals[0]
	for _, v := range vals[1:] {
		if *v.val > *hi.val {
			hi = v
		}
		if *v.val < *lo.val {
			lo = v
		}
	}
	return hi, lo, true
}
