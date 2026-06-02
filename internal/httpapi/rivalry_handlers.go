package httpapi

import (
	"net/http"

	"github.com/MGeovany/rivalo-server/internal/logger"
)

// handleGetRivalries returns aggregated histories per frequent opponent.
//
//	@Summary		Rivalries
//	@Description	Aggregated W/D/L balance, last encounter and averages per frequent opponent.
//	@Tags			sessions
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{array}		session.Rivalry
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/rivalries [get]
func (d Deps) handleGetRivalries(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "rivalries_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	rivalries, err := d.Sessions.GetRivalries(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load rivalries", "rivalries_failed", err, logger.Ref("user", uid))
		return
	}
	writeJSON(w, http.StatusOK, rivalries)
}
