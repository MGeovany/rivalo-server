package httpapi

import (
	"net/http"

	"github.com/MGeovany/rivalo-server/internal/logger"
)

// handleGetPitchStats returns aggregate session stats for one of the user's courts.
//
//	@Summary		Court stats
//	@Description	Matches played, average rating/distance/sprints and last played, for a court the user owns.
//	@Tags			pitches
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"Pitch id"
//	@Success		200	{object}	session.PitchStats
//	@Failure		401	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/pitches/{id}/stats [get]
func (d Deps) handleGetPitchStats(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil || d.Pitches == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "not available", "pitch_stats_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")

	owns, err := d.Pitches.OwnedByUser(r.Context(), uid, id)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load court", "pitch_stats_failed", err, logger.Ref("user", uid))
		return
	}
	if !owns {
		logAndWriteError(w, http.StatusNotFound, "court not found", "pitch_stats_not_found", nil, logger.Ref("user", uid), logger.Ref("pitch", id))
		return
	}

	stats, err := d.Sessions.GetPitchStats(r.Context(), uid, id)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load court stats", "pitch_stats_failed", err, logger.Ref("user", uid))
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
