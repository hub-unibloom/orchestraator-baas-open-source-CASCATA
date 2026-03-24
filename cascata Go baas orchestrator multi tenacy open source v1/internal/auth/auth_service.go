package auth

import (
	"context"
	"fmt"
	"strings"

	"cascata/internal/domain"
	"cascata/internal/service"
	"github.com/golang-jwt/jwt/v5"
)

// AuthService handles key and token validation.
type AuthService struct {
	projectService *service.ProjectService
	sessionManager *SessionManager
}

// NewAuthService creates a new auth manager with system session awareness.
func NewAuthService(projectService *service.ProjectService, sessionManager *SessionManager) *AuthService {
	return &AuthService{
		projectService: projectService,
		sessionManager: sessionManager,
	}
}

// AuthenticateRequest validates incoming headers and returns an AuthContext.
// it identifies a project via `identifier` (Slug or Custom Domain).
func (s *AuthService) AuthenticateRequest(ctx context.Context, identifier, apiKey, authHeader string) (*domain.AuthContext, error) {
	// 1. Resolve Project.
	p, err := s.projectService.Resolve(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("auth.Authenticate: resolve project: %w", err)
	}

	// 2. Resolve Triple-Mode: Service vs Anon vs JWT.
	
	// Mode A: Service Role (God Mode)
	// Phase 25: Support for Multi-Key Rollover
	isPrimary := apiKey == p.ServiceKey
	isLegacy := apiKey != "" && apiKey == p.PreviousServiceKey

	if isPrimary || isLegacy {
		return &domain.AuthContext{
			Mode:         domain.ModeService,
			IdentityType: domain.IdentityAgent, // Service Keys are treated as System Agents for now
			Project:      p,
			ProjectSlug:  p.Slug,
			Role:         "service_role",
		}, nil
	}

	// Mode B: Anon Key (Public restricted) + JWT Check
	if apiKey == p.AnonKey {
		// If there is no Authorization Bearer JWT, we are in Anon mode.
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return &domain.AuthContext{
				Mode:         domain.ModeAnon,
				IdentityType: domain.IdentityResident,
				Project:      p,
				ProjectSlug:  p.Slug,
				Role:         "anon",
			}, nil
		}

		// Mode C: JWT (User Context)
		token := strings.TrimPrefix(authHeader, "Bearer ")
		
		// 1. Try Project-specific Resident JWT logic.
		claims, err := s.verifyJWT(token, p.JWTSecret)
		if err == nil {
			// Phase 10.1 Hardening: Cross-Tenant Protection
			if claims["tenant"] != p.Slug {
				return nil, fmt.Errorf("auth.Authenticate: cross-tenant token violation: token for %v used on %s", claims["tenant"], p.Slug)
			}

			return &domain.AuthContext{
				Mode:         domain.ModeResident,
				IdentityType: domain.IdentityResident,
				Project:      p,
				ProjectSlug:  p.Slug,
				UserID:       fmt.Sprintf("%v", claims["sub"]),
				Role:         fmt.Sprintf("%v", claims["role"]),
				Email:        fmt.Sprintf("%v", claims["email"]),
				HasStepUp:    claims["type"] == "otp_stepup",
				Claims:       claims,
			}, nil
		}

		// 2. Try System-level Membership logic (Passe Livre - Phase 3).
		// If project JWT fails, we check if it's a valid System Member (Worner) token.
		sysCtx, err := s.sessionManager.ValidateSession(ctx, token)
		if err == nil && sysCtx.Role == domain.RoleWorner {
			// Worner has 'God Mode' over any project.
			return &domain.AuthContext{
				Mode:         domain.ModeService,
				IdentityType: domain.IdentityMember,
				Project:      p,
				ProjectSlug:  p.Slug,
				UserID:       sysCtx.UserID,
				Role:         "service_role", // Injected as admin
				Email:        sysCtx.Email,
			}, nil
		}
	}

	return nil, fmt.Errorf("auth.Authenticate: invalid apikey or unauthorized session for project %s", identifier)
}

// verifyJWT validates a JWT using the project-specific secret.
func (s *AuthService) verifyJWT(tokenString, secret string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid or expired resident token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
