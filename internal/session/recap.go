package session

import "time"

// RecapSession is the minimal per-match input for the weekly recap.
type RecapSession struct {
	ID          string
	StartedAt   time.Time
	DistanceM   float64
	Sprints     int
	MatchRating *float64
}

// WeekSummary aggregates the matches of a single ISO week.
type WeekSummary struct {
	MatchCount     int      `json:"match_count"`
	TotalDistanceM float64  `json:"total_distance_m"`
	TotalSprints   int      `json:"total_sprints"`
	AvgRating      *float64 `json:"avg_rating"`
	BestSessionID  *string  `json:"best_session_id"`
}

// WeeklyRecap is this week's summary plus the change vs the previous week.
type WeeklyRecap struct {
	Current          WeekSummary `json:"current"`
	Previous         WeekSummary `json:"previous"`
	DistanceDeltaPct *float64    `json:"distance_delta_pct"`
	RatingDeltaPct   *float64    `json:"rating_delta_pct"`
}

// BuildWeeklyRecap summarizes the current ISO week and compares it with the
// previous one. Pure and deterministic given `now`.
func BuildWeeklyRecap(sessions []RecapSession, now time.Time) WeeklyRecap {
	cur := weekIndex(now)
	var current, previous []RecapSession
	for _, s := range sessions {
		switch weekIndex(s.StartedAt) {
		case cur:
			current = append(current, s)
		case cur - 1:
			previous = append(previous, s)
		}
	}

	c := summarizeWeek(current)
	p := summarizeWeek(previous)

	recap := WeeklyRecap{Current: c, Previous: p}
	if p.TotalDistanceM > 0 {
		v := (c.TotalDistanceM - p.TotalDistanceM) / p.TotalDistanceM * 100
		recap.DistanceDeltaPct = &v
	}
	if c.AvgRating != nil && p.AvgRating != nil && *p.AvgRating > 0 {
		v := (*c.AvgRating - *p.AvgRating) / *p.AvgRating * 100
		recap.RatingDeltaPct = &v
	}
	return recap
}

func summarizeWeek(week []RecapSession) WeekSummary {
	var s WeekSummary
	var ratingSum float64
	var ratingN int
	var bestRating *float64
	var bestDistance float64
	for _, m := range week {
		s.MatchCount++
		s.TotalDistanceM += m.DistanceM
		s.TotalSprints += m.Sprints
		if m.MatchRating != nil {
			ratingSum += *m.MatchRating
			ratingN++
		}
		// Best = highest rating, falling back to longest distance.
		if m.MatchRating != nil {
			if bestRating == nil || *m.MatchRating > *bestRating {
				v := *m.MatchRating
				bestRating = &v
				id := m.ID
				s.BestSessionID = &id
			}
		} else if bestRating == nil && m.DistanceM >= bestDistance {
			bestDistance = m.DistanceM
			id := m.ID
			s.BestSessionID = &id
		}
	}
	if ratingN > 0 {
		v := ratingSum / float64(ratingN)
		s.AvgRating = &v
	}
	return s
}
