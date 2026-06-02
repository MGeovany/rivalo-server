package badge

import (
	"testing"
	"time"
)

var badgeNow = time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)

func find(badges []Badge, key string) Badge {
	for _, b := range badges {
		if b.Key == key {
			return b
		}
	}
	return Badge{}
}

func TestEvaluate_NewlyEarnedAndProgress(t *testing.T) {
	stats := BadgeStats{MatchCount: 12, BestDistanceM: 9000, BestSprints: 22, BestRating: 88, BestStreakWeeks: 5}
	badges, newly := Evaluate(stats, map[string]time.Time{}, badgeNow)

	// first_match, regular, consistent, sprinter earned; veteran/dedicated/long_hauler/elite not.
	if !find(badges, "regular").Earned {
		t.Error("regular should be earned at 12 matches")
	}
	if find(badges, "veteran").Earned {
		t.Error("veteran should not be earned at 12 matches")
	}
	if !find(badges, "sprinter").Earned {
		t.Error("sprinter should be earned at 22 sprints")
	}
	if find(badges, "long_hauler").Earned {
		t.Error("long_hauler should not be earned at 9000 m")
	}
	// Newly-earned should list exactly the earned ones (none pre-earned).
	earnedCount := 0
	for _, b := range badges {
		if b.Earned {
			earnedCount++
		}
	}
	if len(newly) != earnedCount {
		t.Errorf("newly earned = %d, want %d (all fresh)", len(newly), earnedCount)
	}
	// Progress on a pending badge.
	if find(badges, "veteran").Current != 12 || find(badges, "veteran").Target != 50 {
		t.Error("veteran progress should be 12/50")
	}
}

func TestEvaluate_AlreadyEarned_NotNewlyEarned(t *testing.T) {
	earnedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	stats := BadgeStats{MatchCount: 1}
	badges, newly := Evaluate(stats, map[string]time.Time{"first_match": earnedAt}, badgeNow)

	if len(newly) != 0 {
		t.Fatalf("nothing should be newly earned, got %v", newly)
	}
	fm := find(badges, "first_match")
	if !fm.Earned || fm.EarnedAt == nil || !fm.EarnedAt.Equal(earnedAt) {
		t.Errorf("first_match should keep its original earned_at, got %+v", fm)
	}
}
