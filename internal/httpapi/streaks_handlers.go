package httpapi

import (
	"net/http"

	"github.com/MGeovany/rivalo-server/internal/logger"
)

// handleGetStreaks returns the user's weekly streak and special performance streaks.
//
//	@Summary		Streaks
//	@Description	Current/best weekly streak (ISO weeks with a match) plus special performance streaks.
//	@Tags			sessions
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	session.Streaks
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/sessions/streaks [get]
func (d Deps) handleGetStreaks(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "streaks_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	streaks, err := d.Sessions.GetStreaks(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load streaks", "streaks_failed", err, logger.Ref("user", uid))
		return
	}
	writeJSON(w, http.StatusOK, streaks)
}
