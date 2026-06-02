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

// Sample is one point in a session's time series.
type Sample struct {
	TOffsetS int      `json:"t_offset_s"`
	HR       *int     `json:"hr"`
	SpeedKMH *float64 `json:"speed_kmh"`
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
	CreatedAt    time.Time `json:"created_at"`
	// Samples is the time series, populated on detail reads (Get); nil on List.
	Samples []Sample `json:"samples,omitempty"`
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
	Samples      []Sample
}

// Store persists sport sessions.
type Store interface {
	// Create inserts a new session owned by userID and returns the stored row.
	Create(ctx context.Context, userID string, n New) (Session, error)
	// List returns the user's sessions, most recent first.
	List(ctx context.Context, userID string) ([]Session, error)
	// Get returns a single session owned by userID, or ErrNotFound.
	Get(ctx context.Context, userID, id string) (Session, error)
}
