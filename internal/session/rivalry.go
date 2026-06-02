package session

import (
	"sort"
	"time"
)

const MinRivalryMatches = 2

type RivalSession struct {
	Opponent    string
	Outcome     *string
	StartedAt   time.Time
	DistanceM   float64
	Sprints     int
	MatchRating *float64
}

type Rivalry struct {
	Opponent     string    `json:"opponent"`
	MatchCount   int       `json:"match_count"`
	Wins         int       `json:"wins"`
	Draws        int       `json:"draws"`
	Losses       int       `json:"losses"`
	LastPlayedAt time.Time `json:"last_played_at"`
	AvgRating    *float64  `json:"avg_rating"`
	AvgDistanceM *float64  `json:"avg_distance_m"`
	AvgSprints   *float64  `json:"avg_sprints"`
}

func BuildRivalries(sessions []RivalSession) []Rivalry {
	groups := map[string]struct {
		count    int
		wins     int
		draws    int
		losses   int
		last     time.Time
		ratingSum float64
		ratingN   int
		distSum   float64
		sprintSum float64
	}{}

	for _, s := range sessions {
		g := groups[s.Opponent]
		g.count++
		g.distSum += s.DistanceM
		g.sprintSum += float64(s.Sprints)
		if s.MatchRating != nil {
			g.ratingSum += *s.MatchRating
			g.ratingN++
		}
		if s.StartedAt.After(g.last) {
			g.last = s.StartedAt
		}
		if s.Outcome != nil {
			switch *s.Outcome {
			case OutcomeWin:
				g.wins++
			case OutcomeDraw:
				g.draws++
			case OutcomeLoss:
				g.losses++
			}
		}
		groups[s.Opponent] = g
	}

	result := make([]Rivalry, 0, len(groups))
	for name, g := range groups {
		if g.count < MinRivalryMatches {
			continue
		}
		r := Rivalry{
			Opponent:     name,
			MatchCount:   g.count,
			Wins:         g.wins,
			Draws:        g.draws,
			Losses:       g.losses,
			LastPlayedAt: g.last,
		}
		n := float64(g.count)
		r.AvgDistanceM = ptr(g.distSum / n)
		r.AvgSprints = ptr(g.sprintSum / n)
		if g.ratingN > 0 {
			r.AvgRating = ptr(g.ratingSum / float64(g.ratingN))
		}
		result = append(result, r)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].MatchCount > result[j].MatchCount ||
			(result[i].MatchCount == result[j].MatchCount && result[i].Opponent < result[j].Opponent)
	})

	return result
}

func ptr[T any](v T) *T { return &v }
