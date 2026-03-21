package auth

import (
	"context"
	"fmt"
	"strings"

	"cascata/internal/domain"
	"cascata/internal/service"
)

// AuthService handles key and token validation.
type AuthService struct {
	projectService *service.ProjectService
}

// NewAuthService creates a new auth manager.
func NewAuthService(projectService *service.ProjectService) *AuthService {
	return &AuthService{
		projectService: projectService,
	}
}

// AuthenticateRequest validates incoming headers and returns an AuthContext.
// it identifies a project via `projectSlug` identifier (e.g., from URL).
func (s *AuthService) AuthenticateRequest(ctx context.Context, projectSlug, apiKey, authHeader string) (*domain.AuthContext, error) {
	// 1. Resolve Project.
	p, err := s.projectService.Resolve(ctx, projectSlug)
	if err != nil {
		return nil, fmt.Errorf("auth.Authenticate: resolve project: %w", err)
	}

	// 2. Resolve Triple-Mode: Service vs Anon vs JWT.
	
	// Mode A: Service Role (God Mode)
	if apiKey == p.ServiceKey {
		return &domain.AuthContext{
			Mode:        domain.ModeService,
			ProjectSlug: p.Slug,
			Role:        "service_role",
		}, nil
	}

	// Mode B: Anon Key (Public restricted)
	if apiKey == p.AnonKey {
		// If there is no Authorization Bearer JWT, we are in Anon mode.
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return &domain.AuthContext{
				Mode:        domain.ModeAnon,
				ProjectSlug: p.Slug,
				Role:        "anon",
			}, nil
		}

		// Mode C: JWT (User Context)
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := s.verifyJWT(token, p.JWTSecret)
		if err != nil {
			return nil, fmt.Errorf("auth.Authenticate: invalid token: %w", err)
		}

		return &domain.AuthContext{
			Mode:        domain.ModeUser,
			ProjectSlug: p.Slug,
			UserID:      fmt.Sprintf("%v", claims["sub"]),
			Role:        fmt.Sprintf("%v", claims["role"]),
			Email:       fmt.Sprintf("%v", claims["email"]),
			HasStepUp:   claims["type"] == "otp_stepup",
			Claims:      claims,
		}, nil
	}

	return nil, fmt.Errorf("auth.Authenticate: invalid apikey for project %s", projectSlug)
}

// verifyJWT simulates JWT verification (Actual implementation will use a library like golang-jwt).
// For v1.0, we simulate to focus on architectural flow first, as per Phase 3.
func (s *AuthService) verifyJWT(token, secret string) (map[string]interface{}, error) {
	// TODO: Replace with Real JWT Implementation in next iteration of Phase 3.
	// We should avoid external deps until absolutely necessary or instructed.
	if token == "" || secret == "" {
		return nil, fmt.Errorf("invalid token/secret")
	}
	// Simulated response:
	return map[string]interface{}{
		"sub":   "test-uuid",
		"role":  "authenticated",
		"email": "user@test.io",
	}, nil
}
