package session

import (
	"sort"
	"time"
)

// Thresholds for streaks.
const (
	SprintsForStreak       = 20   // sprints in a match to count toward the sprint streak
	SprintStreakThreshold  = 3    // consecutive such matches to "activate" the streak
	RatingStreakThreshold  = 3    // consecutive matches improving Match Rating
	FatigueStreakThreshold = 3    // consecutive structured matches with controlled fatigue
	FatigueControlledMax   = 0.10 // max half-1→half-2 drop to count as "controlled"
)

// StreakSession is the minimal per-match input for streak computation.
type StreakSession struct {
	StartedAt   time.Time
	Sprints     int
	MatchRating *float64
	Structured  bool
	// FatigueControlled is set for structured matches with enough samples per half.
	FatigueControlled *bool
}

// SpecialStreak is a performance-based streak (current leading run of matches).
type SpecialStreak struct {
	Kind      string `json:"kind"`
	Count     int    `json:"count"`
	Threshold int    `json:"threshold"`
	Active    bool   `json:"active"`
}

// Streaks holds the weekly streak plus performance streaks.
type Streaks struct {
	CurrentWeeks int             `json:"current_weeks"`
	BestWeeks    int             `json:"best_weeks"`
	Special      []SpecialStreak `json:"special"`
}

// weekIndex returns a monotonic integer that increments by 1 each ISO week
// (Monday-aligned, UTC). Consecutive Mondays differ by exactly one.
func weekIndex(t time.Time) int {
	t = t.UTC()
	wd := (int(t.Weekday()) + 6) % 7 // Mon=0 … Sun=6
	monday := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -wd)
	return int(monday.Unix() / (7 * 24 * 3600))
}

// BuildStreaks computes the weekly streak (current/best) and special performance
// streaks. Pure and deterministic given `now`.
func BuildStreaks(sessions []StreakSession, now time.Time) Streaks {
	present := map[int]bool{}
	for _, s := range sessions {
		present[weekIndex(s.StartedAt)] = true
	}

	cur := weekIndex(now)
	current := 0
	anchor, hasAnchor := 0, false
	if present[cur] {
		anchor, hasAnchor = cur, true
	} else if present[cur-1] {
		anchor, hasAnchor = cur-1, true
	}
	if hasAnchor {
		for w := anchor; present[w]; w-- {
			current++
		}
	}

	best := longestConsecutive(present)

	// Most-recent-first ordering for leading-run special streaks.
	ordered := append([]StreakSession(nil), sessions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].StartedAt.After(ordered[j].StartedAt)
	})

	return Streaks{
		CurrentWeeks: current,
		BestWeeks:    best,
		Special: []SpecialStreak{
			sprintStreak(ordered),
			ratingStreak(ordered),
			fatigueStreak(ordered),
		},
	}
}

func longestConsecutive(present map[int]bool) int {
	if len(present) == 0 {
		return 0
	}
	weeks := make([]int, 0, len(present))
	for w := range present {
		weeks = append(weeks, w)
	}
	sort.Ints(weeks)
	best, run := 1, 1
	for i := 1; i < len(weeks); i++ {
		if weeks[i] == weeks[i-1]+1 {
			run++
		} else {
			run = 1
		}
		if run > best {
			best = run
		}
	}
	return best
}

func sprintStreak(ordered []StreakSession) SpecialStreak {
	count := 0
	for _, s := range ordered {
		if s.Sprints >= SprintsForStreak {
			count++
		} else {
			break
		}
	}
	return SpecialStreak{Kind: "sprints", Count: count, Threshold: SprintStreakThreshold, Active: count >= SprintStreakThreshold}
}

// ratingStreak counts the leading run of matches whose rating improves over time
// (each more-recent match higher than the next-older one). Only rated matches count.
func ratingStreak(ordered []StreakSession) SpecialStreak {
	var rated []float64
	for _, s := range ordered {
		if s.MatchRating != nil {
			rated = append(rated, *s.MatchRating)
		}
	}
	count := 0
	for i := 0; i < len(rated); i++ {
		if i+1 < len(rated) && rated[i] > rated[i+1] {
			if count == 0 {
				count = 2
			} else {
				count++
			}
		} else {
			break
		}
	}
	return SpecialStreak{Kind: "rating_improving", Count: count, Threshold: RatingStreakThreshold, Active: count >= RatingStreakThreshold}
}

func fatigueStreak(ordered []StreakSession) SpecialStreak {
	count := 0
	for _, s := range ordered {
		if !s.Structured || s.FatigueControlled == nil {
			continue
		}
		if *s.FatigueControlled {
			count++
		} else {
			break
		}
	}
	return SpecialStreak{Kind: "fatigue_controlled", Count: count, Threshold: FatigueStreakThreshold, Active: count >= FatigueStreakThreshold}
}
