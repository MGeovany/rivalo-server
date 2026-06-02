// Package session holds the sport session domain model and storage interface.
package session

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a session does not exist or is not owned by the
// requesting user.
var ErrNotFound = errors.New("session not found")

// Valid session sources.
const (
	SourceManual = "manual"
	SourceWatch  = "watch"
)

// Valid session modes (V2).
const (
	ModeQuick      = "quick"
	ModeStructured = "structured"
	ModeTraining   = "training"
)

// Valid context enums.
const (
	MatchType5aside  = "5-a-side"
	MatchType7aside  = "7-a-side"
	MatchType9aside  = "9-a-side"
	MatchType11aside = "11-a-side"
	MatchTypeOther   = "Other"

	SurfaceNaturalGrass   = "Natural grass"
	SurfaceArtificialTurf = "Artificial turf"
	SurfaceIndoor         = "Indoor"
	SurfaceConcrete       = "Concrete"
	SurfaceOther          = "Other"

	PositionGoalkeeper = "Goalkeeper"
	PositionDefender   = "Defender"
	PositionFullBack   = "Full-back"
	PositionMidfielder = "Midfielder"
	PositionWinger     = "Winger"
	PositionForward    = "Forward"

	MatchTagFriendly = "friendly"
	MatchTagLeague   = "league"
	MatchTagTraining = "training"
)

var ValidMatchTypes = []string{MatchType5aside, MatchType7aside, MatchType9aside, MatchType11aside, MatchTypeOther}
var ValidSurfaces = []string{SurfaceNaturalGrass, SurfaceArtificialTurf, SurfaceIndoor, SurfaceConcrete, SurfaceOther}
var ValidPositions = []string{PositionGoalkeeper, PositionDefender, PositionFullBack, PositionMidfielder, PositionWinger, PositionForward}
var ValidMatchTags = []string{MatchTagFriendly, MatchTagLeague, MatchTagTraining}

// Sample is one point in a session's time series. Half is 1 or 2 for structured
// matches (nil otherwise).
type Sample struct {
	TOffsetS int      `json:"t_offset_s"`
	HR       *int     `json:"hr"`
	SpeedKMH *float64 `json:"speed_kmh"`
	Half     *int     `json:"half,omitempty"`
}

// PathPoint is one GPS sample on the pitch trajectory (V2 session_path).
type PathPoint struct {
	TOffsetS  int     `json:"t_offset_s"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Session is a recorded sport session with its aggregate metrics.
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
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
	Mode         string    `json:"mode"`
	HalftimeOffsetS *int   `json:"halftime_offset_s,omitempty"`
	// Context fields (post-match, optional)
	MatchType     *string  `json:"match_type,omitempty"`
	Surface       *string  `json:"surface,omitempty"`
	Position      *string  `json:"position,omitempty"`
	Result        *string  `json:"result,omitempty"`
	Feeling       *int     `json:"feeling,omitempty"`
	MatchTag      *string  `json:"match_tag,omitempty"`
	PitchID       *string  `json:"pitch_id,omitempty"`
	MatchRating   *float64 `json:"match_rating,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	// Samples is the time series, populated on detail reads (Get); nil on List.
	Samples []Sample `json:"samples,omitempty"`
	// Path is the GPS trajectory, populated on detail reads (Get); nil on List.
	Path []PathPoint `json:"path,omitempty"`
	// FatigueDrop is computed on-read for structured sessions; nil otherwise.
	FatigueDrop *FatigueDrop `json:"fatigue_drop,omitempty"`
}

// New carries the fields needed to create a session.
type New struct {
	StartedAt    time.Time
	EndedAt      time.Time
	DurationS    int
	DistanceM    float64
	HRAvg        *int
	HRMax        *int
	SpeedMaxKMH  *float64
	Sprints      int
	Intensity    *float64
	CaloriesKcal *float64
	Source       string
	Mode         string
	HalftimeOffsetS *int
	Samples      []Sample
	MatchRating  *float64
	PitchID      *string
}

// Store persists sport sessions.
type Store interface {
	// Create inserts a new session owned by userID and returns the stored row.
	Create(ctx context.Context, userID string, n New) (Session, error)
	// List returns the user's sessions, most recent first.
	List(ctx context.Context, userID string) ([]Session, error)
	// Get returns a single session owned by userID, or ErrNotFound.
	Get(ctx context.Context, userID, id string) (Session, error)
	// Update replaces aggregate fields on an owned session.
	Update(ctx context.Context, userID, id string, u Update) (Session, error)
	// UpdateContext patches post-match context fields (does not touch metrics).
	UpdateContext(ctx context.Context, userID, id string, cu ContextUpdate) (Session, error)
	// Delete removes a session owned by userID.
	Delete(ctx context.Context, userID, id string) error
}

// Update carries editable aggregate fields (samples are not modified).
type Update struct {
	StartedAt    time.Time
	EndedAt      time.Time
	DurationS    int
	DistanceM    float64
	HRAvg        *int
	HRMax        *int
	SpeedMaxKMH  *float64
	Sprints      int
	Intensity    *float64
	CaloriesKcal *float64
}

// ContextUpdate carries the post-match context fields that can be PATCHed.
type ContextUpdate struct {
	MatchType *string
	Surface   *string
	Position  *string
	Result    *string
	Feeling   *int
	MatchTag  *string
	PitchID   *string
}
