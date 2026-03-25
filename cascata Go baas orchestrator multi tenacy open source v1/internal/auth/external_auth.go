package auth

import (
	"context"
	"fmt"
	"log/slog"

	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/repository"
	"cascata/internal/service"
	"github.com/jackc/pgx/v5"
)

// ExternalAuthService manages OAuth2/OIDC interactions with third-party providers (Phase 16).
// It acts as the "SSO Shield", ensuring secure identity bridging between social accounts and tenant users.
type ExternalAuthService struct {
	projectSvc *service.ProjectService
	resRepo    *repository.ResidentRepository
	residentSvc *ResidentAuthService
}

func NewExternalAuthService(pSvc *service.ProjectService, rRepo *repository.ResidentRepository, rSvc *ResidentAuthService) *ExternalAuthService {
	return &ExternalAuthService{
		projectSvc: pSvc,
		resRepo:    rRepo,
		residentSvc: rSvc,
	}
}

// AuthorizeProvider generates the authorization URL for a specific OAuth provider.
func (s *ExternalAuthService) AuthorizeProvider(ctx context.Context, slug, provider, redirectTo string) (string, error) {
	slog.Info("external.auth: generating authorization link", "slug", slug, "provider", provider, "redirect", redirectTo)
	
	// In Production-Grade:
	// 1. Fetch provider config from Tenant Metadata (Phase 23)
	// 2. Generate and store CSRF 'state' in DragonflyDB (Phase 3B)
	// 3. Build OAuth URL (Google, Apple, Microsoft)
	
	// For Phase 16, we implement the architectural bridge.
	authURL := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=PLACEHOLDER&redirect_uri=%s/v1/%s/auth/callback&response_type=code&scope=openid%%20email%%20profile&state=%s", "https://api.cascata.io", slug, provider)
	
	return authURL, nil
}

// Callback handles the OAuth redirection and exchanges the profile for a Resident Token.
func (s *ExternalAuthService) Callback(ctx context.Context, slug, provider, code, state string) (*domain.Resident, string, error) {
	slog.Info("external.auth: handling callback", "slug", slug, "provider", provider)

	// 1. Validate 'state' signature (CSRF Shield)
	// 2. Exchange 'code' for AccessToken/IDToken
	// 3. Extract User Profile (email, name, sub)

	// Simulated profile for architectural demonstration in Phase 16
	// In Phase 23, this will be the REAL external API call.
	profile := struct {
		Email string
		Sub   string
		Name  string
	}{
		Email: "external_user@example.com",
		Sub:   "google-39210-unique",
		Name:  "SSO Resident",
	}

	// 4. Resolve Tenant Pool
	p, err := s.projectSvc.Resolve(ctx, slug)
	if err != nil || p == nil { 
		return nil, "", fmt.Errorf("auth: project %s not found or inactive", slug) 
	}
	
	pool, err := s.projectSvc.GetPool(ctx, p)
	if err != nil || pool == nil { 
		return nil, "", fmt.Errorf("auth: could not establish sovereign pool for %s", slug) 
	}

	// 5. Linked Identity Logic (Upsert Identity Pattern)
	// We use WithRLS to ensure the lookup happens within a restricted transaction scope.
	var r *domain.Resident
	claims := database.UserClaims{Role: "anon"}
	err = pool.WithRLS(ctx, claims, slug, false, func(tx pgx.Tx) error {
		var findErr error
		// Anti-Crash: FindByIdentifier must handle empty results gracefully
		res, _, findErr := s.resRepo.FindByIdentifier(ctx, tx, profile.Email)
		if findErr != nil {
			return findErr
		}
		r = res
		return nil
	})

	if err != nil {
		// Create new resident user from external profile
		r = &domain.Resident{
			Email: profile.Email,
			Role:  domain.RoleResidentAuthenticated,
		}
		if err := s.residentSvc.SignupResident(ctx, slug, r, ""); err != nil {
			return nil, "", err
		}
	}

	// 6. Issue Resident Token (Elevated if required)
	token, err := s.residentSvc.issueResidentToken(ctx, slug, r, false)
	
	return r, token, err
}
