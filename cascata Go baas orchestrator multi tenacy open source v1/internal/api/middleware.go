package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"cascata/internal/auth"
	"cascata/internal/domain"
)

// AuthMiddleware wraps handlers with Triple-Mode Auth validation.
type AuthMiddleware struct {
	authService *auth.AuthService
}

// NewAuthMiddleware initializes the security interceptor.
func NewAuthMiddleware(authService *auth.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// EnforceTripleMode intercepts requests and injects the AuthContext.
func (m *AuthMiddleware) EnforceTripleMode(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Identify Project (From Path for now: /v1/:slug/rest/v1/...)
		// Simplified resolution from a header for local testing:
		projectSlug := r.Header.Get("X-Project-Slug")
		if projectSlug == "" {
			// Alternately, check URL path if it follows /v1/<slug>/...
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) > 2 && parts[1] == "v1" {
				projectSlug = parts[2]
			}
		}

		if projectSlug == "" {
			slog.Warn("auth: request rejected - missing project slug")
			http.Error(w, `{"error": "Missing X-Project-Slug or /v1/<slug>/ path"}`, http.StatusBadRequest)
			return
		}

		// 2. Extract Credentials.
		apiKey := r.Header.Get("apikey")
		authHeader := r.Header.Get("Authorization")

		// 3. Authenticate.
		authCtx, err := m.authService.AuthenticateRequest(r.Context(), projectSlug, apiKey, authHeader)
		if err != nil {
			slog.Error("auth: authentication failed", "slug", projectSlug, "error", err)
			http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// 4. Inject into Context.
		ctx := context.WithValue(r.Context(), domain.AuthKey, authCtx)
		
		slog.Debug("auth: identity established", "slug", authCtx.ProjectSlug, "mode", authCtx.Mode, "role", authCtx.Role)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
