// Package profile holds the user profile domain model and storage interface.
package profile

import (
	"context"
	"time"
)

// Profile is a user's profile. ID equals the Supabase auth user id.
type Profile struct {
	ID                string    `json:"id"`
	DisplayName       string    `json:"display_name"`
	PreferredPosition *string   `json:"preferred_position"`
	HeightCM          *int      `json:"height_cm"`
	WeightKG          *float64  `json:"weight_kg"`
	BirthYear         *int      `json:"birth_year"`
	// BirthDate is the full date of birth as "YYYY-MM-DD". birth_year is derived
	// from it; both are returned for backward compatibility.
	BirthDate         *string   `json:"birth_date"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Update carries the mutable fields of a profile.
type Update struct {
	DisplayName       string
	PreferredPosition *string
	HeightCM          *int
	WeightKG          *float64
	BirthYear         *int
	BirthDate         *time.Time
}

// Store persists profiles.
type Store interface {
	// GetOrCreate returns the profile for id, creating a default row the first
	// time the user is seen.
	GetOrCreate(ctx context.Context, id string) (Profile, error)
	// Update applies u to the profile for id and returns the updated row,
	// creating the row if it does not exist yet.
	Update(ctx context.Context, id string, u Update) (Profile, error)
}
