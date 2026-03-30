package models

import (
	"time"

	"gorm.io/gorm"
)

// AuthorizeCode manages temporary authorization sessions for the PKCE flow.
type AuthorizeCode struct {
	gorm.Model
	// Code is the opaque string returned to the client (Step 1).
	Code                string    `gorm:"uniqueIndex;not null"`
	// CodeChallenge is the hashed secret stored in the first step.
	CodeChallenge       string    `gorm:"not null"`
	// CodeChallengeMethod is the transformation method (usually "S256" for SHA256).
	CodeChallengeMethod string    `gorm:"not null"`
	// UserID identifies the human resource owner.
	UserID              uint      `gorm:"not null"`
	// UserRole is the role of the user (e.g., ADMIN, USER).
	UserRole            Role      `gorm:"not null"`
	// ClientID is the ID of the app that initiated the request.
	ClientID            string    `gorm:"not null"`
	// RedirectURI is the specific URI used in the authorize call (Must be registered).
	RedirectURI         string    `gorm:"not null"`
	// State is an optional CSRF protection string.
	State               string
	// ExpiresAt marks when the authorization code will no longer be valid.
	ExpiresAt           time.Time `gorm:"not null"`
	// Used indicates if this code has already been exchanged for a token (Codes MUST be single-use).
	Used                bool      `gorm:"default:false"`
}
