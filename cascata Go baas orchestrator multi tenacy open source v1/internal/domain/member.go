package domain

import "time"

// Member represents a Cascata system operator (Dashboard user).
// Defined in Phase 3 of the Master Plan.
type Member struct {
	ID             string       `json:"id"`
	Email          string       `json:"email"`
	PasswordHash   string       `json:"-"` // Never exposed in JSON
	Role           string       `json:"role"`
	Type           IdentityType `json:"type"` // Standardized to IdentityType (domain.IdentityMember)
	MFAEnabled     bool         `json:"mfa_enabled"`
	MfaCredentials []byte       `json:"-"` // Binary blob for WebAuthn/FIDO2 keys (Phase 27)
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}
