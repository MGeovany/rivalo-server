// Package pitch manages saved playing fields/courts (V2).
package pitch

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("pitch not found")

var ValidTypes = []string{"5-a-side", "7-a-side", "9-a-side", "11-a-side", "Other"}
var ValidSurfaces = []string{"Natural grass", "Artificial turf", "Indoor", "Concrete", "Other"}
var ValidMeasurementMethods = []string{"walk", "camera", "manual"}

// Pitch is a saved pitch owned by a user.
type Pitch struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	Name              string    `json:"name"`
	Latitude          *float64  `json:"latitude,omitempty"`
	Longitude         *float64  `json:"longitude,omitempty"`
	Type              *string   `json:"type,omitempty"`
	Surface           *string   `json:"surface,omitempty"`
	LengthM           *float64  `json:"length_m,omitempty"`
	WidthM            *float64  `json:"width_m,omitempty"`
	MeasurementMethod *string   `json:"measurement_method,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// NewPitch carries the fields to create a pitch.
type NewPitch struct {
	Name              string
	Latitude          *float64
	Longitude         *float64
	Type              *string
	Surface           *string
	LengthM           *float64
	WidthM            *float64
	MeasurementMethod *string
}

// PitchUpdate carries editable fields (nil = not touched).
type PitchUpdate struct {
	Name              *string
	Latitude          *float64
	Longitude         *float64
	Type              *string
	Surface           *string
	LengthM           *float64
	WidthM            *float64
	MeasurementMethod *string
}

// Store persists pitches.
type Store interface {
	Create(ctx context.Context, userID string, n NewPitch) (Pitch, error)
	Get(ctx context.Context, userID, id string) (Pitch, error)
	List(ctx context.Context, userID string) ([]Pitch, error)
	Update(ctx context.Context, userID, id string, u PitchUpdate) (Pitch, error)
	Delete(ctx context.Context, userID, id string) error
	// OwnedByUser reports whether the pitch exists and belongs to userID.
	OwnedByUser(ctx context.Context, userID, id string) (bool, error)
}
