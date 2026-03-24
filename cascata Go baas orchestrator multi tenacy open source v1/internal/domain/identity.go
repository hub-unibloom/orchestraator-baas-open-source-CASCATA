package domain

// IdentityType defines the strict classification of any entity within the Cascata ecosystem.
// No other variants or synonyms are allowed (e.g., Actor, User, Scribe).
type IdentityType string

const (
	IdentityMember   IdentityType = "MEMBER"    // System-level (Worner, Manager, Dev)
	IdentityResident IdentityType = "RESIDENT"  // Tenant-level (Client's end-user)
	IdentityAgent    IdentityType = "AGENT"     // Automation/AI-level (Phantom/Engine)
)

// IdentityContext encapsulates the resolution of who is performing an action.
type IdentityContext struct {
	ID   string       `json:"id"`
	Type IdentityType `json:"type"`
	Role string       `json:"role"`
}

// Cascata Roles for Members (System Context)
const (
	RoleWorner    = "worner"    // Global Admin
	RoleManager   = "manager"   // Tenant Manager
	RoleDeveloper = "developer" // Technical Operator
	RoleAnalyst   = "analyst"   // Read-only Observer
)

// Cascata Roles for Residents (Tenant Context)
const (
	RoleResidentAuthenticated = "authenticated"
	RoleResidentService       = "service"
)
