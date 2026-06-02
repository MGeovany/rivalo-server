package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/MGeovany/rivalo-server/internal/goal"
	"github.com/MGeovany/rivalo-server/internal/logger"
)

type createGoalRequest struct {
	Metric string  `json:"metric"`
	Period string  `json:"period"`
	Target float64 `json:"target"`
}

type updateGoalRequest struct {
	Metric   *string  `json:"metric,omitempty"`
	Period   *string  `json:"period,omitempty"`
	Target   *float64 `json:"target,omitempty"`
	Archived *bool    `json:"archived,omitempty"`
}

// computeProgress takes goal store + session store and returns a goal with
// progress filled in (and newly achieved goals marked).
func computeProgress(ctx context.Context, d Deps, uid string, g goal.Goal, now time.Time) (goal.Goal, error) {
	if d.Sessions == nil {
		return g, nil
	}
	list, err := d.Sessions.List(ctx, uid)
	if err != nil {
		return g, err
	}

	var sessions []goal.GoalSession
	for _, s := range list {
		sessions = append(sessions, goal.GoalSession{
			StartedAt:   s.StartedAt,
			DistanceM:   s.DistanceM,
			Sprints:     s.Sprints,
			MatchRating: s.MatchRating,
		})
	}

	progress, newlyAchieved := goal.CalculateProgress(g, sessions, now)
	g.Progress = progress
	if newlyAchieved && d.Goals != nil {
		if err := d.Goals.Achieve(ctx, uid, g.ID, now); err != nil {
			logger.Error("goal_achieve_failed", logger.Ref("user", uid), logger.Ref("goal", g.ID), logger.SafeErr(err))
		} else {
			g.AchievedAt = &now
		}
	}
	return g, nil
}

// handleCreateGoal creates a new personal goal.
//
//	@Summary		Create goal
//	@Description	Create a personal goal on a session metric over a period.
//	@Tags			goals
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		createGoalRequest	true	"Goal fields"
//	@Success		201		{object}	goal.Goal
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		503		{object}	errorResponse
//	@Router			/v1/goals [post]
func (d Deps) handleCreateGoal(w http.ResponseWriter, r *http.Request) {
	if d.Goals == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "goals are not available", "goals_unavailable", nil)
		return
	}

	var req createGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logAndWriteError(w, http.StatusBadRequest, "invalid request body", "goals_bad_request", err)
		return
	}

	if !contains(goal.ValidMetrics, req.Metric) {
		logAndWriteError(w, http.StatusBadRequest, "invalid metric: "+req.Metric, "goals_invalid_metric", nil)
		return
	}
	if !contains(goal.ValidPeriods, req.Period) {
		logAndWriteError(w, http.StatusBadRequest, "invalid period: "+req.Period, "goals_invalid_period", nil)
		return
	}
	if req.Target <= 0 {
		logAndWriteError(w, http.StatusBadRequest, "target must be > 0", "goals_invalid_target", nil)
		return
	}

	uid := userID(r.Context())
	created, err := d.Goals.Create(r.Context(), uid, goal.NewGoal{
		Metric: req.Metric,
		Period: req.Period,
		Target: req.Target,
	})
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not create goal", "goals_create_failed", err, logger.Ref("user", uid))
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// handleListGoals returns all active goals with computed progress.
//
//	@Summary		List goals
//	@Description	List active personal goals with progress toward each target.
//	@Tags			goals
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{array}		goal.Goal
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/goals [get]
func (d Deps) handleListGoals(w http.ResponseWriter, r *http.Request) {
	if d.Goals == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "goals are not available", "goals_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	goals, err := d.Goals.List(r.Context(), uid)
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load goals", "goals_list_failed", err, logger.Ref("user", uid))
		return
	}

	now := time.Now().UTC()
	for i, g := range goals {
		computed, err := computeProgress(r.Context(), d, uid, g, now)
		if err != nil {
			logger.Error("goal_progress_failed", logger.Ref("user", uid), logger.Ref("goal", g.ID), logger.SafeErr(err))
			goals[i].Progress = 0
			continue
		}
		goals[i] = computed
	}

	writeJSON(w, http.StatusOK, goals)
}

// handleGetGoal returns one goal with computed progress.
//
//	@Summary		Get goal
//	@Description	Get a personal goal by ID with computed progress.
//	@Tags			goals
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	goal.Goal
//	@Failure		404	{object}	errorResponse
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/goals/{id} [get]
func (d Deps) handleGetGoal(w http.ResponseWriter, r *http.Request) {
	if d.Goals == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "goals are not available", "goals_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	g, err := d.Goals.Get(r.Context(), uid, id)
	if errors.Is(err, goal.ErrNotFound) {
		logAndWriteError(w, http.StatusNotFound, "goal not found", "goals_not_found", nil, logger.Ref("goal", id))
		return
	}
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not load goal", "goals_get_failed", err, logger.Ref("user", uid), logger.Ref("goal", id))
		return
	}

	computed, err := computeProgress(r.Context(), d, uid, g, time.Now().UTC())
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not compute progress", "goals_progress_failed", err, logger.Ref("user", uid), logger.Ref("goal", id))
		return
	}

	writeJSON(w, http.StatusOK, computed)
}

// handleUpdateGoal updates editable fields on a goal.
//
//	@Summary		Update goal
//	@Description	Update metric, period, target, or archive a goal.
//	@Tags			goals
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		updateGoalRequest	true	"Fields to update"
//	@Success		200		{object}	goal.Goal
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		503		{object}	errorResponse
//	@Router			/v1/goals/{id} [patch]
func (d Deps) handleUpdateGoal(w http.ResponseWriter, r *http.Request) {
	if d.Goals == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "goals are not available", "goals_unavailable", nil)
		return
	}

	var req updateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logAndWriteError(w, http.StatusBadRequest, "invalid request body", "goals_bad_request", err)
		return
	}

	if req.Metric != nil && !contains(goal.ValidMetrics, *req.Metric) {
		logAndWriteError(w, http.StatusBadRequest, "invalid metric", "goals_invalid_metric", nil)
		return
	}
	if req.Period != nil && !contains(goal.ValidPeriods, *req.Period) {
		logAndWriteError(w, http.StatusBadRequest, "invalid period", "goals_invalid_period", nil)
		return
	}
	if req.Target != nil && *req.Target <= 0 {
		logAndWriteError(w, http.StatusBadRequest, "target must be > 0", "goals_invalid_target", nil)
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	updated, err := d.Goals.Update(r.Context(), uid, id, goal.GoalUpdate{
		Metric:   req.Metric,
		Period:   req.Period,
		Target:   req.Target,
		Archived: req.Archived,
	})
	if errors.Is(err, goal.ErrNotFound) {
		logAndWriteError(w, http.StatusNotFound, "goal not found", "goals_not_found", nil, logger.Ref("goal", id))
		return
	}
	if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not update goal", "goals_update_failed", err, logger.Ref("user", uid), logger.Ref("goal", id))
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// handleDeleteGoal deletes a goal.
//
//	@Summary		Delete goal
//	@Description	Delete a personal goal by ID.
//	@Tags			goals
//	@Security		BearerAuth
//	@Success		204
//	@Failure		404	{object}	errorResponse
//	@Failure		401	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/v1/goals/{id} [delete]
func (d Deps) handleDeleteGoal(w http.ResponseWriter, r *http.Request) {
	if d.Goals == nil {
		logAndWriteError(w, http.StatusServiceUnavailable, "goals are not available", "goals_unavailable", nil)
		return
	}

	uid := userID(r.Context())
	id := r.PathValue("id")
	if err := d.Goals.Delete(r.Context(), uid, id); errors.Is(err, goal.ErrNotFound) {
		logAndWriteError(w, http.StatusNotFound, "goal not found", "goals_not_found", nil, logger.Ref("goal", id))
		return
	} else if err != nil {
		logAndWriteError(w, http.StatusInternalServerError, "could not delete goal", "goals_delete_failed", err, logger.Ref("user", uid), logger.Ref("goal", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}


