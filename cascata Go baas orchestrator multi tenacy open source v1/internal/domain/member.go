package domain

import "time"

// MemberRole represents the authorization level of a Cascata member.
type MemberRole string

const (
	RoleWorner    MemberRole = "worner"    // Global Admin
	RoleManager   MemberRole = "manager"   // Tenant Manager
	RoleDeveloper MemberRole = "developer" // Technical Operator
	RoleAnalyst   MemberRole = "analyst"   // Read-only Observer
	RoleAgent     MemberRole = "agent"     // AI/Automation Agent
)

// MemberType distinguishes between human operators and AI agents.
type MemberType string

const (
	TypeHuman MemberType = "human"
	TypeAgent MemberType = "agent"
)

// Member represents a Cascata system operator (Dashboard user).
// Defined in Phase 3 of the Master Plan.
type Member struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // Never exposed in JSON
	Role         MemberRole `json:"role"`
	Type         MemberType `json:"type"`
	MFAEnabled     bool       `json:"mfa_enabled"`
	MfaCredentials []byte     `json:"-"` // Binary blob for WebAuthn/FIDO2 keys (Phase 27)
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// AuditLog represents an immutable record of a management action.
type AuditLog struct {
	ID         string     `json:"id"`
	MemberID   string     `json:"member_id"`
	MemberType MemberType `json:"member_type"`
	Action     string     `json:"action"`
	EntityType string     `json:"entity_type"`
	EntityID   string     `json:"entity_id,omitempty"`
	TenantSlug string     `json:"tenant_slug,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time  `json:"created_at"`
}
