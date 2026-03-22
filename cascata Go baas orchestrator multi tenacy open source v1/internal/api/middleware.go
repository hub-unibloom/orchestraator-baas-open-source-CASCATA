package api

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"cascata/internal/auth"
	"cascata/internal/domain"
	"cascata/internal/ratelimit"
)

// RateLimitMiddleware handles early-rejection of banned IPs and quota enforcement.
type RateLimitMiddleware struct {
	engine *ratelimit.AdaptiveEngine
}

func NewRateLimitMiddleware(engine *ratelimit.AdaptiveEngine) *RateLimitMiddleware {
	return &RateLimitMiddleware{engine: engine}
}

// MemberAuthMiddleware protects system-level management endpoints.
type MemberAuthMiddleware struct {
	sessionSvc *auth.SessionManager
}

func NewMemberAuthMiddleware(sessionSvc *auth.SessionManager) *MemberAuthMiddleware {
	return &MemberAuthMiddleware{sessionSvc: sessionSvc}
}

// EnforceMemberSession ensures the request comes from an authenticated system operator.
// Restricts access to the Dashboard and configuration tools.
func (m *MemberAuthMiddleware) EnforceMemberSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error": "AUTHENTICATION_REQUIRED"}`, http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		authCtx, err := m.sessionSvc.ValidateSession(token)
		if err != nil {
			slog.Warn("member.auth: session validation failed", "error", err)
			http.Error(w, `{"error": "SESSION_EXPIRED_OR_INVALID"}`, http.StatusUnauthorized)
			return
		}

		// Inject AuthContext into Go context for system handlers.
		ctx := context.WithValue(r.Context(), domain.AuthKey, authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// EnforceRateLimit performs IP strike checks and operation-weighted restriction.
func (m *RateLimitMiddleware) EnforceRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}

		// 1. IP Ban Check
		if banned, ttl := m.engine.IsIPBanned(r.Context(), ip); banned {
			slog.Warn("ratelimit: rejecting banned IP", "ip", ip, "ttl", ttl)
			http.Error(w, `{"error": "IP_BANNED_SECURITY_VIOLATION"}`, http.StatusForbidden)
			return
		}

		// Successive checks (like quota) will happen AFTER Auth context is established,
		// because we need the project slug to know which quota bucket to use.
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware wraps handlers with Triple-Mode Auth validation.
type AuthMiddleware struct {
	authService *auth.AuthService
	ratelimit   *ratelimit.AdaptiveEngine
}

// NewAuthMiddleware initializes the security interceptor with adaptive rate limiting.
func NewAuthMiddleware(authService *auth.AuthService, rl *ratelimit.AdaptiveEngine) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		ratelimit:   rl,
	}
}

// EnforceTripleMode intercepts requests and injects the AuthContext.
// It is the heart of Multi-Tenancy resolution at the entry point.
func (m *AuthMiddleware) EnforceTripleMode(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Resolve Project Identifier (Multi-Head Detection)
		var identifier string

		// Strategy A: Explicit Header (Highest priority for SDKs)
		identifier = r.Header.Get("X-Project-Slug")

		// Strategy B: Host Header (Custom Domains)
		if identifier == "" {
			host := r.Host // e.g. api.clientapp.com:8080 or clientapp.com
			if strings.Contains(host, ":") {
				host, _, _ = net.SplitHostPort(host)
			}
			identifier = host
		}

		// Strategy C: URL Path Fallback (/v1/:slug/...)
		if identifier == "" || identifier == "localhost" || identifier == "127.0.0.1" {
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) > 2 && parts[1] == "v1" {
				identifier = parts[2]
			}
		}

		if identifier == "" {
			slog.Warn("auth: tenant resolution failed - no identifier found")
			slog.Debug("request details", "host", r.Host, "path", r.URL.Path)
			http.Error(w, `{"error": "Tenant Resolution Failed: Specify X-Project-Slug or use a Registered Domain"}`, http.StatusBadRequest)
			return
		}

		// 2. Adaptive Rate Limiting & Firewall
		// This happens AFTER resolution so we have a tenant-specific bucket.
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}

		// Check Quota (Defaults used if project-specific quota not yet loaded from DB)
		// We use a safe default: 100 requests per minute with no nerf.
		quotaCfg := ratelimit.Config{Limit: 100, WindowSeconds: 60, SpeedPctOnNerf: 10}
		allowed, err := m.ratelimit.Check(r.Context(), identifier, ip, r.Method, quotaCfg, false)
		if err != nil || !allowed {
			slog.Warn("ratelimit: quota exceeded or system check failed", "id", identifier, "ip", ip)
			http.Error(w, `{"error": "Too Many Requests: Quota Exceeded"}`, http.StatusTooManyRequests)
			return
		}

		// 3. Extract Credentials
		apiKey := r.Header.Get("apikey")
		authHeader := r.Header.Get("Authorization")

		// 4. Authenticate and Establish Project Context
		authCtx, err := m.authService.AuthenticateRequest(r.Context(), identifier, apiKey, authHeader)
		if err != nil {
			slog.Error("auth: project resolution or key validation failed", "id", identifier, "err", err)
			
			// Record a security strike for invalid keys (IP Strikes)
			_ = m.ratelimit.ReportIPStrike(r.Context(), ip)
			
			http.Error(w, `{"error": "Unauthorized: Project not found or invalid key"}`, http.StatusUnauthorized)
			return
		}

		// 5. Inject into Context for downstream handlers (Realtime, Storage, SQL)
		ctx := context.WithValue(r.Context(), domain.AuthKey, authCtx)
		
		slog.Debug("auth: project bound", "slug", authCtx.ProjectSlug, "mode", authCtx.Mode)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
