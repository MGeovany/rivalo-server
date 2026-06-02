package httpapi

import (
	"net/http"

	"github.com/MGeovany/rivalo-server/internal/logger"
)

// handleGetInsights returns aggregated session stats for the authenticated user.
//
//	@Summary		Session insights
//	@Description	Returns totals, averages, and per-context breakdowns (match_type, surface, position).
//	@Tags			sessions
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	session.SessionInsights
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/sessions/insights [get]
func (d Deps) handleGetInsights(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "insights_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	ins, err := d.Sessions.GetInsights(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load insights", "insights_failed", err, logger.Ref("user", uid))
		return
	}
	writeJSON(w, http.StatusOK, ins)
}
