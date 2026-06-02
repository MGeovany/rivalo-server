package httpapi

import (
	"net/http"

	"github.com/MGeovany/rivalo-server/internal/logger"
)

// handleGetPositionInsights returns a cautious, physical-only comparison of the
// user's positions. It never declares a "best" position.
//
//	@Summary		Position insights
//	@Description	Physical averages per position with neutral comparisons. Requires at least 3 sessions in each of 2+ positions; otherwise has_enough_data is false.
//	@Tags			sessions
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	session.PositionInsights
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/sessions/position-insights [get]
func (d Deps) handleGetPositionInsights(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "position_insights_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	insights, err := d.Sessions.GetPositionInsights(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load position insights", "position_insights_failed", err, logger.Ref("user", uid))
		return
	}
	writeJSON(w, http.StatusOK, insights)
}
