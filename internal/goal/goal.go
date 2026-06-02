// Package goal holds personal goal domain logic — user-defined targets on session
// metrics with on-read progress calculation.
package goal

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("goal not found")

var ValidMetrics = []string{"distance", "matches", "sprints", "rating"}
var ValidPeriods = []string{"week", "month"}

// GoalSession is the minimal per-match input for progress calculation.
type GoalSession struct {
	StartedAt   time.Time
	DistanceM   float64
	Sprints     int
	MatchRating *float64
}

// Goal is a user-defined target with computed progress.
type Goal struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Metric      string     `json:"metric"`
	Period      string     `json:"period"`
	Target      float64    `json:"target"`
	CreatedAt   time.Time  `json:"created_at"`
	AchievedAt  *time.Time `json:"achieved_at,omitempty"`
	Archived    bool       `json:"archived"`
	Progress    float64    `json:"progress"`
}

// NewGoal carries the fields needed to create a goal.
type NewGoal struct {
	Metric string
	Period string
	Target float64
}

// GoalUpdate carries editable fields for an existing goal.
type GoalUpdate struct {
	Metric *string
	Period *string
	Target *float64
	Archived *bool
}

// weekIndex returns a monotonic integer per ISO week (same as session package).
func weekIndex(t time.Time) int {
	t = t.UTC()
	wd := (int(t.Weekday()) + 6) % 7
	monday := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -wd)
	return int(monday.Unix() / (7 * 24 * 3600))
}

// monthIndex returns a monotonic integer per calendar month.
func monthIndex(t time.Time) int {
	t = t.UTC()
	return t.Year()*12 + int(t.Month())
}

// CalculateProgress computes the current value of a goal's metric within its
// period, based on the given sessions. It returns the progress value and whether
// the goal is newly achieved (progress >= target and was not already achieved).
func CalculateProgress(g Goal, sessions []GoalSession, now time.Time) (progress float64, newlyAchieved bool) {
	var idx int
	switch g.Period {
	case "week":
		idx = weekIndex(now)
	case "month":
		idx = monthIndex(now)
	default:
		return 0, false
	}

	var (
		sumDist, sumSprints float64
		count               int
		ratingSum           float64
		ratingN             int
	)
	for _, s := range sessions {
		var sidx int
		switch g.Period {
		case "week":
			sidx = weekIndex(s.StartedAt)
		case "month":
			sidx = monthIndex(s.StartedAt)
		}
		if sidx != idx {
			continue
		}
		count++
		sumDist += s.DistanceM
		sumSprints += float64(s.Sprints)
		if s.MatchRating != nil {
			ratingSum += *s.MatchRating
			ratingN++
		}
	}

	switch g.Metric {
	case "matches":
		progress = float64(count)
	case "distance":
		progress = sumDist
	case "sprints":
		progress = sumSprints
	case "rating":
		if ratingN > 0 {
			progress = ratingSum / float64(ratingN)
		}
	}

	newlyAchieved = g.AchievedAt == nil && progress >= g.Target
	return
}

// Store persists personal goals.
type Store interface {
	Create(ctx context.Context, userID string, n NewGoal) (Goal, error)
	List(ctx context.Context, userID string) ([]Goal, error)
	Get(ctx context.Context, userID, id string) (Goal, error)
	Update(ctx context.Context, userID, id string, u GoalUpdate) (Goal, error)
	Delete(ctx context.Context, userID, id string) error
	// Achieve sets achieved_at if the goal is not already achieved.
	Achieve(ctx context.Context, userID, id string, at time.Time) error
}
