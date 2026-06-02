// Package badge defines the achievement catalog and grant persistence.
package badge

import (
	"context"
	"time"
)

// BadgeStats are the aggregate inputs a badge condition can depend on.
type BadgeStats struct {
	MatchCount      int
	BestDistanceM   float64
	BestSprints     int
	BestRating      float64
	BestStreakWeeks int
}

// Badge is a catalog entry with the user's progress toward it.
type Badge struct {
	Key         string     `json:"key"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Target      float64    `json:"target"`
	Current     float64    `json:"current"`
	Earned      bool       `json:"earned"`
	EarnedAt    *time.Time `json:"earned_at,omitempty"`
}

type definition struct {
	key, title, desc string
	target           float64
	value            func(BadgeStats) float64
}

// catalog is the fixed set of achievements (code-defined, ordered).
var catalog = []definition{
	{"first_match", "First match", "Play your first match.", 1, func(s BadgeStats) float64 { return float64(s.MatchCount) }},
	{"regular", "Regular", "Play 10 matches.", 10, func(s BadgeStats) float64 { return float64(s.MatchCount) }},
	{"veteran", "Veteran", "Play 50 matches.", 50, func(s BadgeStats) float64 { return float64(s.MatchCount) }},
	{"consistent", "Consistent", "Reach a 4-week streak.", 4, func(s BadgeStats) float64 { return float64(s.BestStreakWeeks) }},
	{"dedicated", "Dedicated", "Reach a 12-week streak.", 12, func(s BadgeStats) float64 { return float64(s.BestStreakWeeks) }},
	{"long_hauler", "Long hauler", "Cover 10 km in a single match.", 10000, func(s BadgeStats) float64 { return s.BestDistanceM }},
	{"sprinter", "Sprinter", "Hit 20 sprints in a match.", 20, func(s BadgeStats) float64 { return float64(s.BestSprints) }},
	{"elite", "Elite engine", "Score a 90+ Match Rating.", 90, func(s BadgeStats) float64 { return s.BestRating }},
}

// Evaluate returns every badge with progress, honoring already-earned grants and
// reporting which ones were newly earned this evaluation (for persistence).
func Evaluate(stats BadgeStats, earned map[string]time.Time, now time.Time) (badges []Badge, newlyEarned []string) {
	badges = make([]Badge, 0, len(catalog))
	for _, d := range catalog {
		cur := d.value(stats)
		b := Badge{Key: d.key, Title: d.title, Description: d.desc, Target: d.target, Current: cur}
		if at, ok := earned[d.key]; ok {
			at := at
			b.Earned = true
			b.EarnedAt = &at
		} else if cur >= d.target {
			at := now
			b.Earned = true
			b.EarnedAt = &at
			newlyEarned = append(newlyEarned, d.key)
		}
		badges = append(badges, b)
	}
	return badges, newlyEarned
}

// Store persists earned badges.
type Store interface {
	// Earned returns the user's earned badge keys with the time each was granted.
	Earned(ctx context.Context, userID string) (map[string]time.Time, error)
	// Grant records a badge as earned (idempotent — no-op if already present).
	Grant(ctx context.Context, userID, key string, at time.Time) error
}
