package httpapi

import (
	"errors"
	"net/http"

	"github.com/MGeovany/rivalo-server/internal/logger"
	"github.com/MGeovany/rivalo-server/internal/pitch"
)

// createPitchRequest is the JSON body for POST /v1/pitches.
type createPitchRequest struct {
	Name              string   `json:"name"`
	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
	Type              *string  `json:"type"`
	Surface           *string  `json:"surface"`
	LengthM           *float64 `json:"length_m"`
	WidthM            *float64 `json:"width_m"`
	HeadingDeg        *float64 `json:"heading_deg"`
	MeasurementMethod *string  `json:"measurement_method"`
	Indoor            *bool    `json:"indoor"`
	Notes             *string  `json:"notes"`
}

// updatePitchRequest is the JSON body for PUT /v1/pitches/{id}.
type updatePitchRequest struct {
	Name              *string  `json:"name"`
	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
	Type              *string  `json:"type"`
	Surface           *string  `json:"surface"`
	LengthM           *float64 `json:"length_m"`
	WidthM            *float64 `json:"width_m"`
	HeadingDeg        *float64 `json:"heading_deg"`
	MeasurementMethod *string  `json:"measurement_method"`
	Indoor            *bool    `json:"indoor"`
	Notes             *string  `json:"notes"`
}

// handleCreatePitch stores a new pitch for the authenticated user.
//
//	@Summary		Create a pitch
//	@Tags			pitches
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		createPitchRequest	true	"Pitch payload"
//	@Success		201		{object}	pitch.Pitch
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		503		{object}	errorResponse
//	@Router			/v1/pitches [post]
func (d Deps) handleCreatePitch(w http.ResponseWriter, r *http.Request) {
	if d.Pitches == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "pitches are not available", "pitch_create_unavailable", nil)
		return
	}

	var req createPitchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		logAndWriteError(w, http.StatusBadRequest, "invalid JSON body", "pitch_create_rejected", err, "reason", "invalid_json")
		return
	}

	p, msg := req.validate()
	if msg != "" {
		logAndWriteError(w, http.StatusBadRequest, msg, "pitch_create_rejected", nil, "reason", "validation_failed")
		return
	}

	uid := userID(r.Context())
	created, err := d.Pitches.Create(r.Context(), uid, p)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not create pitch", "pitch_create_failed", err, logger.Ref("user", uid))
		return
	}
	logger.Info("pitch_create_ok", logger.Ref("user", uid), logger.Ref("pitch", created.ID))
	writeJSON(w, http.StatusCreated, created)
}

// handleListPitches returns the user's saved pitches.
//
//	@Summary		List my pitches
//	@Tags			pitches
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{array}		pitch.Pitch
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/pitches [get]
func (d Deps) handleListPitches(w http.ResponseWriter, r *http.Request) {
	if d.Pitches == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "pitches are not available", "pitch_list_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	pitches, err := d.Pitches.List(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not list pitches", "pitch_list_failed", err, logger.Ref("user", uid))
		return
	}
	writeJSON(w, http.StatusOK, pitches)
}

// handleGetPitch returns one pitch by id.
//
//	@Summary		Get a pitch
//	@Tags			pitches
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"Pitch id"
//	@Success		200	{object}	pitch.Pitch
//	@Failure		401	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/pitches/{id} [get]
func (d Deps) handleGetPitch(w http.ResponseWriter, r *http.Request) {
	if d.Pitches == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "pitches are not available", "pitch_get_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	found, err := d.Pitches.Get(r.Context(), uid, id)
	if errors.Is(err, pitch.ErrNotFound) {
		logAndWriteError(w, http.StatusNotFound, "pitch not found", "pitch_get_not_found", err, logger.Ref("user", uid), logger.Ref("pitch", id))
		return
	}
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load pitch", "pitch_get_failed", err, logger.Ref("user", uid), logger.Ref("pitch", id))
		return
	}
	writeJSON(w, http.StatusOK, found)
}

// handleUpdatePitch updates an existing pitch.
//
//	@Summary		Update a pitch
//	@Tags			pitches
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string				true	"Pitch id"
//	@Param			body	body	updatePitchRequest	true	"Pitch update (partial)"
//	@Success		200		{object}	pitch.Pitch
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		503		{object}	errorResponse
//	@Router			/v1/pitches/{id} [put]
func (d Deps) handleUpdatePitch(w http.ResponseWriter, r *http.Request) {
	if d.Pitches == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "pitches are not available", "pitch_update_unavailable", nil)
		return
	}

	var req updatePitchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		logAndWriteError(w, http.StatusBadRequest, "invalid JSON body", "pitch_update_rejected", err, "reason", "invalid_json")
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	updated, err := d.Pitches.Update(r.Context(), uid, id, req.toUpdate())
	if errors.Is(err, pitch.ErrNotFound) {
		logAndWriteError(w, http.StatusNotFound, "pitch not found", "pitch_update_not_found", err, logger.Ref("user", uid), logger.Ref("pitch", id))
		return
	}
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not update pitch", "pitch_update_failed", err, logger.Ref("user", uid), logger.Ref("pitch", id))
		return
	}
	logger.Info("pitch_update_ok", logger.Ref("user", uid), logger.Ref("pitch", id))
	writeJSON(w, http.StatusOK, updated)
}

// handleDeletePitch deletes a pitch.
//
//	@Summary		Delete a pitch
//	@Tags			pitches
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Pitch id"
//	@Success		204	{object}	noContent
//	@Failure		401	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/pitches/{id} [delete]
func (d Deps) handleDeletePitch(w http.ResponseWriter, r *http.Request) {
	if d.Pitches == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "pitches are not available", "pitch_delete_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	err := d.Pitches.Delete(r.Context(), uid, id)
	if errors.Is(err, pitch.ErrNotFound) {
		logAndWriteError(w, http.StatusNotFound, "pitch not found", "pitch_delete_not_found", err, logger.Ref("user", uid), logger.Ref("pitch", id))
		return
	}
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not delete pitch", "pitch_delete_failed", err, logger.Ref("user", uid), logger.Ref("pitch", id))
		return
	}
	logger.Info("pitch_delete_ok", logger.Ref("user", uid), logger.Ref("pitch", id))
	w.WriteHeader(http.StatusNoContent)
}

// validate checks the create-pitch request.
func (req createPitchRequest) validate() (pitch.NewPitch, string) {
	if req.Name == "" {
		return pitch.NewPitch{}, "name is required"
	}
	if req.Latitude != nil && (*req.Latitude < -90 || *req.Latitude > 90) {
		return pitch.NewPitch{}, "latitude must be between -90 and 90"
	}
	if req.Longitude != nil && (*req.Longitude < -180 || *req.Longitude > 180) {
		return pitch.NewPitch{}, "longitude must be between -180 and 180"
	}
	if req.Type != nil && !contains(pitch.ValidTypes, *req.Type) {
		return pitch.NewPitch{}, "type is not a valid value"
	}
	if req.Surface != nil && !contains(pitch.ValidSurfaces, *req.Surface) {
		return pitch.NewPitch{}, "surface is not a valid value"
	}
	if req.MeasurementMethod != nil && !contains(pitch.ValidMeasurementMethods, *req.MeasurementMethod) {
		return pitch.NewPitch{}, "measurement_method is not a valid value"
	}
	if req.LengthM != nil && *req.LengthM < 0 {
		return pitch.NewPitch{}, "length_m must be zero or positive"
	}
	if req.WidthM != nil && *req.WidthM < 0 {
		return pitch.NewPitch{}, "width_m must be zero or positive"
	}
	if req.HeadingDeg != nil && (*req.HeadingDeg < 0 || *req.HeadingDeg >= 360) {
		return pitch.NewPitch{}, "heading_deg must be between 0 and 360"
	}
	return pitch.NewPitch{
		Name:              req.Name,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		Type:              req.Type,
		Surface:           req.Surface,
		LengthM:           req.LengthM,
		WidthM:            req.WidthM,
		HeadingDeg:        req.HeadingDeg,
		MeasurementMethod: req.MeasurementMethod,
		Indoor:            req.Indoor,
		Notes:             req.Notes,
	}, ""
}

func (req updatePitchRequest) toUpdate() pitch.PitchUpdate {
	return pitch.PitchUpdate{
		Name:              req.Name,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		Type:              req.Type,
		Surface:           req.Surface,
		LengthM:           req.LengthM,
		WidthM:            req.WidthM,
		HeadingDeg:        req.HeadingDeg,
		MeasurementMethod: req.MeasurementMethod,
		Indoor:            req.Indoor,
		Notes:             req.Notes,
	}
}
