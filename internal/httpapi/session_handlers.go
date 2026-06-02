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
	Mode            string `json:"mode"`
	HalftimeOffsetS *int   `json:"halftime_offset_s"`
	Samples      []sampleRequest `json:"samples"`
}

// sampleRequest is one time-series point in a create-session payload.
type sampleRequest struct {
	TOffsetS int      `json:"t_offset_s"`
	HR       *int     `json:"hr"`
	SpeedKMH *float64 `json:"speed_kmh"`
	Half     *int     `json:"half"`
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
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "session_create_unavailable", nil)
		return
	}

	var req createSessionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		logAndWriteError(w, http.StatusBadRequest, "invalid JSON body", "session_create_rejected", err, "reason", "invalid_json")
		return
	}

	newSession, msg := req.validate()
	if msg != "" {
		logAndWriteError(w, http.StatusBadRequest, msg, "session_create_rejected", nil, "reason", "validation_failed")
		return
	}

	uid := userID(r.Context())

	// Compute match_rating if we have HR samples and a profile/birth_year.
	rating := computeMatchRating(r, d, uid, newSession)

	newSession.MatchRating = rating

	created, err := d.Sessions.Create(r.Context(), uid, newSession)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not create session", "session_create_failed", err, logger.Ref("user", uid))
		return
	}
	logger.Info("session_create_ok", logger.Ref("user", uid), logger.Ref("session", created.ID))
	writeJSON(w, http.StatusCreated, created)
}

// computeMatchRating calculates the Edwards TRIMP-based match rating.
// It uses the user's birth_year for HRmax (Tanaka formula) or falls back to
// the session's observed max HR.
func computeMatchRating(r *http.Request, d Deps, uid string, newSession session.New) *float64 {
	if newSession.HRMax == nil || len(newSession.Samples) == 0 {
		return nil
	}
	hrMax := *newSession.HRMax

	if d.Profiles != nil {
		profile, err := d.Profiles.GetOrCreate(r.Context(), uid)
		if err == nil && profile.BirthYear != nil {
			if estimated := session.HRmaxByAge(*profile.BirthYear, time.Now().Year()); estimated > hrMax {
				hrMax = estimated
			}
		}
	}

	return session.CalculateMatchRating(newSession.Samples, hrMax, newSession.DurationS)
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
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "session_list_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	sessions, err := d.Sessions.List(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load sessions", "session_list_failed", err, logger.Ref("user", uid))
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
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "session_get_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	found, err := d.Sessions.Get(r.Context(), uid, id)
	if errors.Is(err, session.ErrNotFound) {
		logAndWriteError(w, http.StatusNotFound, "session not found", "session_get_not_found", err, logger.Ref("user", uid), logger.Ref("session", id))
		return
	}
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load session", "session_get_failed", err, logger.Ref("user", uid), logger.Ref("session", id))
		return
	}

	// Compute Fatigue Drop for structured sessions on read.
	if found.Mode == session.ModeStructured && found.HalftimeOffsetS != nil {
		hrMax := resolveHRMax(r, d, uid, found.HRMax)
		found.FatigueDrop = session.ComputeFatigueDrop(found.Mode, found.Samples, found.HalftimeOffsetS, hrMax)
	}

	writeJSON(w, http.StatusOK, found)
}

// resolveHRMax returns the best available HRmax for a user: from the profile's
// birth_year (Tanaka formula) if available, or the observed max HR from the
// session as fallback.
func resolveHRMax(r *http.Request, d Deps, uid string, observedMax *int) int {
	if d.Profiles != nil {
		profile, err := d.Profiles.GetOrCreate(r.Context(), uid)
		if err == nil && profile.BirthYear != nil {
			estimated := session.HRmaxByAge(*profile.BirthYear, time.Now().Year())
			if observedMax != nil && estimated > *observedMax {
				return estimated
			}
			if observedMax == nil {
				return estimated
			}
		}
	}
	if observedMax != nil {
		return *observedMax
	}
	return 220 // absolute fallback (unlikely to be reached in practice)
}

// updateSessionRequest is the JSON body accepted by PUT /v1/sessions/{id}.
type updateSessionRequest struct {
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
}

func (d Deps) handleUpdateSession(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "session_update_unavailable", nil)
		return
	}

	var req updateSessionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		logAndWriteError(w, http.StatusBadRequest, "invalid JSON body", "session_update_rejected", err, "reason", "invalid_json")
		return
	}

	update, msg := req.validate()
	if msg != "" {
		logAndWriteError(w, http.StatusBadRequest, msg, "session_update_rejected", nil, "reason", "validation_failed")
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	updated, err := d.Sessions.Update(r.Context(), uid, id, update)
	if errors.Is(err, session.ErrNotFound) {
		logAndWriteError(w, http.StatusNotFound, "session not found", "session_update_not_found", err, logger.Ref("user", uid), logger.Ref("session", id))
		return
	}
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not update session", "session_update_failed", err, logger.Ref("user", uid), logger.Ref("session", id))
		return
	}
	logger.Info("session_update_ok", logger.Ref("user", uid), logger.Ref("session", id))
	writeJSON(w, http.StatusOK, updated)
}

// patchSessionRequest is the JSON body accepted by PATCH /v1/sessions/{id}.
type patchSessionRequest struct {
	MatchType *string `json:"match_type"`
	Surface   *string `json:"surface"`
	Position  *string `json:"position"`
	Result    *string `json:"result"`
	Feeling   *int    `json:"feeling"`
	MatchTag  *string `json:"match_tag"`
	PitchID   *string `json:"pitch_id"`
}

func (d Deps) handlePatchSessionContext(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "session_patch_unavailable", nil)
		return
	}

	var req patchSessionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		logAndWriteError(w, http.StatusBadRequest, "invalid JSON body", "session_patch_rejected", err, "reason", "invalid_json")
		return
	}

	cu, msg := req.validate()
	if msg != "" {
		logAndWriteError(w, http.StatusBadRequest, msg, "session_patch_rejected", nil, "reason", "validation_failed")
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	updated, err := d.Sessions.UpdateContext(r.Context(), uid, id, cu)
	if errors.Is(err, session.ErrNotFound) {
		logAndWriteError(w, http.StatusNotFound, "session not found", "session_patch_not_found", err, logger.Ref("user", uid), logger.Ref("session", id))
		return
	}
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not update session context", "session_patch_failed", err, logger.Ref("user", uid), logger.Ref("session", id))
		return
	}
	logger.Info("session_patch_context_ok", logger.Ref("user", uid), logger.Ref("session", id))
	writeJSON(w, http.StatusOK, updated)
}

func (d Deps) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if d.Sessions == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "sessions are not available", "session_delete_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	err := d.Sessions.Delete(r.Context(), uid, id)
	if errors.Is(err, session.ErrNotFound) {
		logAndWriteError(w, http.StatusNotFound, "session not found", "session_delete_not_found", err, logger.Ref("user", uid), logger.Ref("session", id))
		return
	}
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not delete session", "session_delete_failed", err, logger.Ref("user", uid), logger.Ref("session", id))
		return
	}
	logger.Info("session_delete_ok", logger.Ref("user", uid), logger.Ref("session", id))
	w.WriteHeader(http.StatusNoContent)
}

func (req updateSessionRequest) validate() (session.Update, string) {
	if req.StartedAt.IsZero() || req.EndedAt.IsZero() {
		return session.Update{}, "started_at and ended_at are required"
	}
	if req.EndedAt.Before(req.StartedAt) {
		return session.Update{}, "ended_at must not be before started_at"
	}
	if req.DurationS < 0 {
		return session.Update{}, "duration_s must be zero or positive"
	}
	if req.DistanceM < 0 {
		return session.Update{}, "distance_m must be zero or positive"
	}
	if req.Sprints < 0 {
		return session.Update{}, "sprints must be zero or positive"
	}
	if !inRangeInt(req.HRAvg, 20, 260) || !inRangeInt(req.HRMax, 20, 260) {
		return session.Update{}, "heart rate values must be between 20 and 260"
	}
	if req.SpeedMaxKMH != nil && *req.SpeedMaxKMH < 0 {
		return session.Update{}, "speed_max_kmh must be zero or positive"
	}
	if !inRangeFloat(req.Intensity, 0, 100) {
		return session.Update{}, "intensity must be between 0 and 100"
	}
	if req.CaloriesKcal != nil && *req.CaloriesKcal < 0 {
		return session.Update{}, "calories_kcal must be zero or positive"
	}
	return session.Update{
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
	}, ""
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

	// Mode: default to quick when unset; only structured carries a halftime.
	mode := req.Mode
	if mode == "" {
		mode = session.ModeQuick
	}
	switch mode {
	case session.ModeQuick, session.ModeStructured, session.ModeTraining:
	default:
		return session.New{}, "mode must be 'quick', 'structured' or 'training'"
	}
	if req.HalftimeOffsetS != nil {
		if mode != session.ModeStructured {
			return session.New{}, "halftime_offset_s is only valid for a structured match"
		}
		if *req.HalftimeOffsetS < 0 || *req.HalftimeOffsetS > req.DurationS {
			return session.New{}, "halftime_offset_s must be within the session duration"
		}
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
		if s.Half != nil && *s.Half != 1 && *s.Half != 2 {
			return session.New{}, "sample half must be 1 or 2"
		}
		samples = append(samples, session.Sample{TOffsetS: s.TOffsetS, HR: s.HR, SpeedKMH: s.SpeedKMH, Half: s.Half})
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
		Mode:         mode,
		HalftimeOffsetS: req.HalftimeOffsetS,
		Samples:      samples,
	}, ""
}

func (req patchSessionRequest) validate() (session.ContextUpdate, string) {
	if req.Feeling != nil && (*req.Feeling < 1 || *req.Feeling > 5) {
		return session.ContextUpdate{}, "feeling must be between 1 and 5"
	}
	if req.Result != nil && len(*req.Result) > 500 {
		return session.ContextUpdate{}, "result must be at most 500 characters"
	}

	cu := session.ContextUpdate{
		MatchType: req.MatchType,
		Surface:   req.Surface,
		Position:  req.Position,
		Result:    req.Result,
		Feeling:   req.Feeling,
		MatchTag:  req.MatchTag,
		PitchID:   req.PitchID,
	}

	if cu.MatchType != nil {
		if !contains(session.ValidMatchTypes, *cu.MatchType) {
			return session.ContextUpdate{}, "match_type is not a valid value"
		}
	}
	if cu.Surface != nil {
		if !contains(session.ValidSurfaces, *cu.Surface) {
			return session.ContextUpdate{}, "surface is not a valid value"
		}
	}
	if cu.Position != nil {
		if !contains(session.ValidPositions, *cu.Position) {
			return session.ContextUpdate{}, "position is not a valid value"
		}
	}
	if cu.MatchTag != nil {
		if !contains(session.ValidMatchTags, *cu.MatchTag) {
			return session.ContextUpdate{}, "match_tag is not a valid value"
		}
	}

	return cu, ""
}

func contains(list []string, item string) bool {
	for _, l := range list {
		if l == item {
			return true
		}
	}
	return false
}

// inRangeInt reports whether an optional int is absent or within [min, max].
func inRangeInt(v *int, min, max int) bool {
	return v == nil || (*v >= min && *v <= max)
}

// inRangeFloat reports whether an optional float is absent or within [min, max].
func inRangeFloat(v *float64, min, max float64) bool {
	return v == nil || (*v >= min && *v <= max)
}
