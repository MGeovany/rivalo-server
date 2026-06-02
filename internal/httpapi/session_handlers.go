package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/MGeovany/rivalo-server/internal/logger"
	"github.com/MGeovany/rivalo-server/internal/session"
)

// createSessionRequest is the JSON body accepted by POST /v1/sessions.
type createSessionRequest struct {
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	DurationS    int       `json:"duration_s"`
	DistanceM    float64   `json:"distance_m"`
	HRAvg        *int      `json:"hr_avg"`
	HRMax        *int      `json:"hr_max"`
	SpeedMaxKMH  *float64  `json:"speed_max_kmh"`
	Sprints      int       `json:"sprints"`
	Intensity    *float64  `json:"intensity"`
	CaloriesKcal *float64  `json:"calories_kcal"`
	Source       string    `json:"source"`
	Samples      []sampleRequest `json:"samples"`
}

// sampleRequest is one time-series point in a create-session payload.
type sampleRequest struct {
	TOffsetS int      `json:"t_offset_s"`
	HR       *int     `json:"hr"`
	SpeedKMH *float64 `json:"speed_kmh"`
}

// handleCreateSession stores a new sport session for the authenticated user.
//
//	@Summary		Create a session
//	@Description	Stores a new sport session owned by the authenticated user.
//	@Tags			sessions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		createSessionRequest	true	"Session payload"
//	@Success		201		{object}	session.Session
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		503		{object}	errorResponse
//	@Router			/v1/sessions [post]
func (d Deps) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		writeError(w, http.StatusServiceUnavailable, "sessions are not available")
		return
	}

	var req createSessionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		logger.Warn("session_create_rejected", "reason", "invalid_json")
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	newSession, msg := req.validate()
	if msg != "" {
		logger.Warn("session_create_rejected", "reason", "validation_failed")
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	uid := userID(r.Context())
	created, err := d.Sessions.Create(r.Context(), uid, newSession)
	if err != nil {
		logger.Error("session_create_failed", logger.Ref("user", uid), logger.SafeErr(err))
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	logger.Info("session_create_ok", logger.Ref("user", uid), logger.Ref("session", created.ID))
	writeJSON(w, http.StatusCreated, created)
}

// handleListSessions returns the authenticated user's sessions, most recent first.
//
//	@Summary		List my sessions
//	@Tags			sessions
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{array}		session.Session
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/sessions [get]
func (d Deps) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		writeError(w, http.StatusServiceUnavailable, "sessions are not available")
		return
	}

	uid := userID(r.Context())
	sessions, err := d.Sessions.List(r.Context(), uid)
	if err != nil {
		logger.Error("session_list_failed", logger.Ref("user", uid), logger.SafeErr(err))
		writeError(w, http.StatusInternalServerError, "could not load sessions")
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

// handleGetSession returns one of the authenticated user's sessions by id.
//
//	@Summary		Get a session
//	@Tags			sessions
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"Session id"
//	@Success		200	{object}	session.Session
//	@Failure		401	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/sessions/{id} [get]
func (d Deps) handleGetSession(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		writeError(w, http.StatusServiceUnavailable, "sessions are not available")
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	found, err := d.Sessions.Get(r.Context(), uid, id)
	if errors.Is(err, session.ErrNotFound) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		logger.Error("session_get_failed", logger.Ref("user", uid), logger.SafeErr(err))
		writeError(w, http.StatusInternalServerError, "could not load session")
		return
	}
	writeJSON(w, http.StatusOK, found)
}

// validate checks the request and returns the session to create, or a non-empty
// message describing the first validation failure.
func (req createSessionRequest) validate() (session.New, string) {
	switch req.Source {
	case session.SourceManual, session.SourceWatch:
	default:
		return session.New{}, "source must be 'manual' or 'watch'"
	}
	if req.StartedAt.IsZero() || req.EndedAt.IsZero() {
		return session.New{}, "started_at and ended_at are required"
	}
	if req.EndedAt.Before(req.StartedAt) {
		return session.New{}, "ended_at must not be before started_at"
	}
	if req.DurationS < 0 {
		return session.New{}, "duration_s must be zero or positive"
	}
	if req.DistanceM < 0 {
		return session.New{}, "distance_m must be zero or positive"
	}
	if req.Sprints < 0 {
		return session.New{}, "sprints must be zero or positive"
	}
	if !inRangeInt(req.HRAvg, 20, 260) || !inRangeInt(req.HRMax, 20, 260) {
		return session.New{}, "heart rate values must be between 20 and 260"
	}
	if req.SpeedMaxKMH != nil && *req.SpeedMaxKMH < 0 {
		return session.New{}, "speed_max_kmh must be zero or positive"
	}
	if !inRangeFloat(req.Intensity, 0, 100) {
		return session.New{}, "intensity must be between 0 and 100"
	}
	if req.CaloriesKcal != nil && *req.CaloriesKcal < 0 {
		return session.New{}, "calories_kcal must be zero or positive"
	}

	const maxSamples = 5000
	if len(req.Samples) > maxSamples {
		return session.New{}, "too many samples (max 5000)"
	}
	samples := make([]session.Sample, 0, len(req.Samples))
	for _, s := range req.Samples {
		if s.TOffsetS < 0 {
			return session.New{}, "sample t_offset_s must be zero or positive"
		}
		samples = append(samples, session.Sample{TOffsetS: s.TOffsetS, HR: s.HR, SpeedKMH: s.SpeedKMH})
	}

	return session.New{
		StartedAt:    req.StartedAt,
		EndedAt:      req.EndedAt,
		DurationS:    req.DurationS,
		DistanceM:    req.DistanceM,
		HRAvg:        req.HRAvg,
		HRMax:        req.HRMax,
		SpeedMaxKMH:  req.SpeedMaxKMH,
		Sprints:      req.Sprints,
		Intensity:    req.Intensity,
		CaloriesKcal: req.CaloriesKcal,
		Source:       req.Source,
		Samples:      samples,
	}, ""
}

// inRangeInt reports whether an optional int is absent or within [min, max].
func inRangeInt(v *int, min, max int) bool {
	return v == nil || (*v >= min && *v <= max)
}

// inRangeFloat reports whether an optional float is absent or within [min, max].
func inRangeFloat(v *float64, min, max float64) bool {
	return v == nil || (*v >= min && *v <= max)
}
