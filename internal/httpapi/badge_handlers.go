package httpapi

import (
	"net/http"
	"time"

	"github.com/MGeovany/rivalo-server/internal/badge"
	"github.com/MGeovany/rivalo-server/internal/logger"
)

type badgesResponse struct {
	Badges []badge.Badge `json:"badges"`
}

// handleGetBadges evaluates the badge catalog for the user, grants newly-earned
// badges idempotently and returns each badge with progress.
//
//	@Summary		Badges
//	@Description	Earned and pending achievement badges with progress.
//	@Tags			sessions
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	badgesResponse
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/badges [get]
func (d Deps) handleGetBadges(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil || d.Badges == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "badges are not available", "badges_unavailable", nil)
		return
	}

	uid := userID(r.Context())

	metrics, err := d.Sessions.GetBadgeMetrics(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load badges", "badges_failed", err, logger.Ref("user", uid))
		return
	}
	streaks, err := d.Sessions.GetStreaks(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load badges", "badges_failed", err, logger.Ref("user", uid))
		return
	}
	earned, err := d.Badges.Earned(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load badges", "badges_failed", err, logger.Ref("user", uid))
		return
	}

	stats := badge.BadgeStats{
		MatchCount:      metrics.MatchCount,
		BestDistanceM:   metrics.BestDistanceM,
		BestSprints:     metrics.BestSprints,
		BestRating:      metrics.BestRating,
		BestStreakWeeks: streaks.BestWeeks,
	}

	now := time.Now().UTC()
	badges, newlyEarned := badge.Evaluate(stats, earned, now)
	for _, key := range newlyEarned {
		if err := d.Badges.Grant(r.Context(), uid, key, now); err != nil {
			logger.Error("badge_grant_failed", logger.Ref("user", uid), logger.Ref("badge", key), logger.SafeErr(err))
		}
	}

	writeJSON(w, http.StatusOK, badgesResponse{Badges: badges})
}
