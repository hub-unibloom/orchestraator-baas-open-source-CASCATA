package domain

import "context"

// Identity types for Cascata.
const (
	IdentityTenantUser     = "tenant.user"    // Resident of the apartment
	IdentityCascataMember  = "cascata.member" // Manager of the building
	IdentityCascataAgent   = "cascata.agent"  // Programmatic AI/Automation agent
)

// Auth Mode types.
const (
	ModeService  = "service"  // Manager mode (Member access)
	ModeAnon     = "anon"     // Guest access
	ModeResident = "resident" // Resident access (RLS restricted)
)

// AuthContext carries the identity and mode throughout a request.
type AuthContext struct {
	Mode         string
	IdentityType string // cascata.member or tenant.user
	Project      *Project
	ProjectSlug  string
	UserID       string
	Role         string
	Email       string
	HasStepUp   bool
	Claims      map[string]interface{}
}

// ContextKey is the type for context keys to avoid collisions.
type ContextKey string

const AuthKey ContextKey = "cascata.auth"

// FromContext retrieves the AuthContext from a Go context.
func FromContext(ctx context.Context) (*AuthContext, bool) {
	val := ctx.Value(AuthKey)
	if val == nil {
		return nil, false
	}
	ac, ok := val.(*AuthContext)
	return ac, ok
}
