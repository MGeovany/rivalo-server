package httpapi

import (
	"net/http"

	"github.com/MGeovany/rivalo-server/internal/logger"
)

// handleGetRecords returns the authenticated user's personal bests.
//
//	@Summary		Personal records
//	@Description	Returns the best value per metric and the session it was achieved in.
//	@Tags			sessions
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	session.PersonalRecords
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/sessions/records [get]
func (d Deps) handleGetRecords(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "records_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	records, err := d.Sessions.GetPersonalRecords(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load records", "records_failed", err, logger.Ref("user", uid))
		return
	}
	writeJSON(w, http.StatusOK, records)
}
