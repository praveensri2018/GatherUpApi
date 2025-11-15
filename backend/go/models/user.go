/* Place: backend/go/models/user.go */
package models

import "time"

// User represents the profile row from dbo.users (auth credentials are stored in user_credentials).
type User struct {
	ID               string  `json:"id"`
	MobileNumber     *string `json:"mobile_number,omitempty"`
	MobileNormalized *string `json:"mobile_normalized,omitempty"`
	CountryCode      *string `json:"country_code,omitempty"`
	DisplayName      *string `json:"display_name,omitempty"`
	AvatarURL        *string `json:"avatar_url,omitempty"`
	Bio              *string `json:"bio,omitempty"`
	Email            *string `json:"email,omitempty"`
	Username         *string `json:"username,omitempty"`

	Latitude          *float64   `json:"latitude,omitempty"`
	Longitude         *float64   `json:"longitude,omitempty"`
	LocationUpdatedAt *time.Time `json:"location_updated_at,omitempty"`

	DateOfBirth *time.Time `json:"date_of_birth,omitempty"`
	Gender      *string    `json:"gender,omitempty"`

	IsMobileVerified bool       `json:"is_mobile_verified,omitempty"`
	MobileVerifiedAt *time.Time `json:"mobile_verified_at,omitempty"`
	IsEmailVerified  *bool      `json:"is_email_verified,omitempty"`
	IsActive         bool       `json:"is_active,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	// internal flags, not serialized
	IsDeleted bool `json:"-"`
	// rv rowversion was in schema; keep it out of JSON
	Rv []byte `json:"-"`
}
