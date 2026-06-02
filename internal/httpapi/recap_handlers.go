package httpapi

import (
	"net/http"

	"github.com/MGeovany/rivalo-server/internal/logger"
)

// handleGetWeeklyRecap returns the current-week summary and the change vs last week.
//
//	@Summary		Weekly recap
//	@Description	Aggregates of the current ISO week (matches, distance, sprints, rating, best match) and deltas vs the previous week.
//	@Tags			sessions
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	session.WeeklyRecap
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/recap/weekly [get]
func (d Deps) handleGetWeeklyRecap(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "recap_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	recap, err := d.Sessions.GetWeeklyRecap(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load recap", "recap_failed", err, logger.Ref("user", uid))
		return
	}
	writeJSON(w, http.StatusOK, recap)
}
