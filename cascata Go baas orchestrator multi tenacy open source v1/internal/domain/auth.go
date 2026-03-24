package domain

import "context"

// Auth Mode types for Cascata (RLS & Access Context).
const (
	ModeService  = "SERVICE"   // Dashboard/Management mode
	ModeAnon     = "ANON"      // Public/Guest access
	ModeResident = "RESIDENT"  // Tenant end-user access (RLS)
)

// AuthContext carries the identity and mode throughout a request.
type AuthContext struct {
	Mode         string
	IdentityType IdentityType // MEMBER, RESIDENT, AGENT
	Project      *Project
	ProjectSlug  string
	UserID       string
	Role         string
	Email        string
	HasStepUp    bool
	Claims       map[string]interface{}
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
