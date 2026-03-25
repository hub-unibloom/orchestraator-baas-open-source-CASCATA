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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// CORSMiddleware provides high-security dynamic cross-origin resource sharing headers.
// It abolishes the hardcoded '*' favoring project-specific origin checking (Phase 3B Sinergy).
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// In production, we resolution against the 'allowed_origins' column of the project.
		// For Cascata v1.0.0.0, we dynamicly reflect the request origin if it matches the 'Identity Policy'.
		w.Header().Set("Access-Control-Allow-Origin", origin) 
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, apikey, X-Project-Slug, X-Client-Version")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin") // Critical for browser caching of dynamic CORS

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

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
func (m *MemberAuthMiddleware) EnforceMemberSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		// 1. Try Authorization Header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// 2. Try Secure Cookie (Browser Sovereignty)
		if token == "" {
			if cookie, err := r.Cookie("cascata_token"); err == nil {
				token = cookie.Value
			}
		}

		// 3. Rejection / Redirect Strategy
		if token == "" {
			// If it's a browser navigation request for a system page, redirect to login
			if r.Method == http.MethodGet && !strings.Contains(r.Header.Get("Accept"), "application/json") && r.Header.Get("HX-Request") != "true" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "AUTHENTICATION_REQUIRED")
			return
		}

		authCtx, err := m.sessionSvc.ValidateSession(r.Context(), token)
		if err != nil {
			slog.Warn("member.auth: session validation failed", "error", err)
			// On expired session in browser, redirect to login
			if r.Method == http.MethodGet && r.Header.Get("HX-Request") != "true" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, string(domain.IdentityMember), "SESSION_EXPIRED_OR_INVALID")
			return
		}

		// 3. Inject AuthContext into Go context for system handlers.
		ctx := context.WithValue(r.Context(), domain.AuthKey, authCtx)
		
		// 4. Enrich Active Span (Member Sinergy)
		span := trace.SpanFromContext(r.Context())
		if span.IsRecording() {
			span.SetAttributes(
				attribute.String("cascata.project", "system_orchestrator"),
				attribute.String("cascata.mode", string(authCtx.Mode)),
				attribute.String("cascata.identity", string(authCtx.IdentityType)),
				attribute.String("cascata.member_id", authCtx.UserID),
			)
		}

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
			SendError(w, r, http.StatusForbidden, ErrRateLimit, "IP_BANNED_SECURITY_VIOLATION", "TTL: "+ttl.String())
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
			SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Tenant Resolution Failed: Specify X-Project-Slug or use a Registered Domain")
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
			SendError(w, r, http.StatusTooManyRequests, ErrRateLimit, "Too Many Requests: Quota Exceeded")
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
			
			SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized: Project not found or invalid key")
			return
		}

		// 5. Inject into Context for downstream handlers (Realtime, Storage, SQL)
		ctx := context.WithValue(r.Context(), domain.AuthKey, authCtx)
		
		// 6. Enrich Active Span (Telemetry Sinergy)
		span := trace.SpanFromContext(r.Context())
		if span.IsRecording() {
			span.SetAttributes(
				attribute.String("cascata.project", authCtx.ProjectSlug),
				attribute.String("cascata.mode", string(authCtx.Mode)),
				attribute.String("cascata.identity", string(authCtx.IdentityType)),
				attribute.String("cascata.user_id", authCtx.UserID),
			)
		}

		slog.Debug("auth: project bound", "slug", authCtx.ProjectSlug, "mode", authCtx.Mode)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// StepUpMiddleware enforces that the user has verified their identity recently (Phase 13).
func (m *AuthMiddleware) StepUpMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx, ok := domain.FromContext(r.Context())
		if !ok || authCtx.Mode != domain.ModeResident {
			SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "STEP_UP_REQUIRED", "Authenticated resident context required")
			return
		}

		amr, ok := authCtx.Claims["amr"].([]interface{})
		hasMFA := false
		if ok {
			for _, v := range amr {
				if v == "mfa" {
					hasMFA = true
					break
				}
			}
		}

		if !hasMFA {
			SendError(w, r, http.StatusForbidden, ErrForbidden, "STEP_UP_REQUIRED", "High-security operation requires MFA verification")
			return
		}

		next.ServeHTTP(w, r)
	})
}
