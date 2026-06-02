package httpapi

import (
	"net/http"
	"strings"

	"github.com/MGeovany/rivalo-server/internal/logger"
	"github.com/MGeovany/rivalo-server/internal/profile"
)

// updateProfileRequest is the JSON body accepted by PUT /v1/me.
type updateProfileRequest struct {
	DisplayName       string   `json:"display_name"`
	PreferredPosition *string  `json:"preferred_position"`
	HeightCM          *int     `json:"height_cm"`
	WeightKG          *float64 `json:"weight_kg"`
}

// handleGetMe returns the authenticated user's profile, creating it on first access.
//
//	@Summary		Get my profile
//	@Description	Returns the authenticated user's profile, creating a default one on first access.
//	@Tags			profile
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	profile.Profile
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/me [get]
func (d Deps) handleGetMe(w http.ResponseWriter, r *http.Request) {
	if d.Profiles == nil {
		logger.Warn("profile_get_unavailable")
		writeError(w, http.StatusServiceUnavailable, "profiles are not available")
		return
	}

	uid := userID(r.Context())
	p, err := d.Profiles.GetOrCreate(r.Context(), uid)
	if err != nil {
		logger.Error("profile_get_failed", logger.Ref("user", uid), logger.SafeErr(err))
		writeError(w, http.StatusInternalServerError, "could not load profile")
		return
	}
	logger.Info("profile_get_ok", logger.Ref("user", uid))
	writeJSON(w, http.StatusOK, p)
}

// handleUpdateMe updates the authenticated user's profile.
//
//	@Summary		Update my profile
//	@Description	Updates the authenticated user's profile fields.
//	@Tags			profile
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		updateProfileRequest	true	"Profile fields"
//	@Success		200		{object}	profile.Profile
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		503		{object}	errorResponse
//	@Router			/v1/me [put]
func (d Deps) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	if d.Profiles == nil {
		logger.Warn("profile_update_unavailable")
		writeError(w, http.StatusServiceUnavailable, "profiles are not available")
		return
	}

	var req updateProfileRequest
	if err := decodeJSON(w, r, &req); err != nil {
		logger.Warn("profile_update_rejected", "reason", "invalid_json")
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	update, msg := req.validate()
	if msg != "" {
		logger.Warn("profile_update_rejected", "reason", "validation_failed")
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	uid := userID(r.Context())
	p, err := d.Profiles.Update(r.Context(), uid, update)
	if err != nil {
		logger.Error("profile_update_failed", logger.Ref("user", uid), logger.SafeErr(err))
		writeError(w, http.StatusInternalServerError, "could not update profile")
		return
	}
	logger.Info("profile_update_ok", logger.Ref("user", uid))
	writeJSON(w, http.StatusOK, p)
}

// validate checks the request and returns the sanitized update, or a non-empty
// message describing the first validation failure.
func (req updateProfileRequest) validate() (profile.Update, string) {
	name := strings.TrimSpace(req.DisplayName)
	if name == "" {
		return profile.Update{}, "display_name is required"
	}
	if len(name) > 50 {
		return profile.Update{}, "display_name must be at most 50 characters"
	}

	update := profile.Update{DisplayName: name}

	if req.PreferredPosition != nil {
		pos := strings.TrimSpace(*req.PreferredPosition)
		if len(pos) > 30 {
			return profile.Update{}, "preferred_position must be at most 30 characters"
		}
		if pos != "" {
			update.PreferredPosition = &pos
		}
	}

	if req.HeightCM != nil {
		if *req.HeightCM < 50 || *req.HeightCM > 260 {
			return profile.Update{}, "height_cm must be between 50 and 260"
		}
		update.HeightCM = req.HeightCM
	}

	if req.WeightKG != nil {
		if *req.WeightKG < 20 || *req.WeightKG > 400 {
			return profile.Update{}, "weight_kg must be between 20 and 400"
		}
		update.WeightKG = req.WeightKG
	}

	return update, ""
}
