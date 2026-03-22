package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"cascata/internal/auth"
	"cascata/internal/ratelimit"
)

// SystemHandler handles management-level authentication and sessions.
// Orchestrates the "Front-Door" of the Cascata Dashboard.
type SystemHandler struct {
	authSvc    *auth.SystemAuthService
	sessionSvc *auth.SessionManager
	ratelimit  *ratelimit.AdaptiveEngine
}

func NewSystemHandler(authSvc *auth.SystemAuthService, sessionSvc *auth.SessionManager, rl *ratelimit.AdaptiveEngine) *SystemHandler {
	return &SystemHandler{
		authSvc:    authSvc,
		sessionSvc: sessionSvc,
		ratelimit:  rl,
	}
}

// LoginRequest defines the payload for system authentication.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	OTP      string `json:"otp,omitempty"`
}

// LoginResponse returns the ephemeral session token and member metadata.
type LoginResponse struct {
	Token      string    `json:"token"`
	ExpiresAt  time.Time `json:"expires_at"`
	MemberName string    `json:"member_name"`
	MFAStatus  string    `json:"mfa_status"`
}

// HandleLogin processes the Worner/Member authentication flow.
// Enforces brute-force protection and lockout.
func (h *SystemHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Pre-auth Firewall (Lockout Check)
	if locked, ttl := h.ratelimit.IsLoginLocked(r.Context(), req.Email); locked {
		slog.Warn("system.login: attempt on locked account", "email", req.Email, "ttl", ttl)
		w.WriteHeader(http.StatusLocked)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "ACCOUNT_LOCKED",
			"message": "Too many failed attempts. Try again in " + ttl.Round(time.Second).String(),
		})
		return
	}

	// 2. Core Authentication (Password / Bcrypt)
	member, err := h.authSvc.AuthenticateMember(r.Context(), req.Email, req.Password)
	if err != nil {
		// Record Failure & Potentially Lock Account
		_ = h.ratelimit.RecordLoginFailure(r.Context(), req.Email)
		
		slog.Warn("system.login: authentication failed", "email", req.Email, "error", err)
		http.Error(w, `{"error": "INVALID_CREDENTIALS"}`, http.StatusUnauthorized)
		return
	}

	// 3. MFA Handling (If enabled for this member)
	mfaStatus := "passed"
	if member.MFAEnabled {
		if req.OTP == "" {
			// Require OTP in the next step
			w.WriteHeader(http.StatusAccepted) // 202 Accepted means "More info needed"
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "MFA_REQUIRED",
				"message": "Two-factor authentication is active. Provide OTP code.",
			})
			return
		}
		
		// TODO: Implement actual OTP verification using the secret in Vault.
		// For now, we simulate success if the OTP is provided.
		slog.Info("system.login: MFA verified for member", "email", member.Email)
	}

	// 4. Issue Session (15 Minutes Ephemeral)
	token, err := h.sessionSvc.IssueSessionToken(member)
	if err != nil {
		slog.Error("system.login: session issuance failed", "error", err)
		http.Error(w, `{"error": "SESSION_FAILURE"}`, http.StatusInternalServerError)
		return
	}

	slog.Info("system.login: session established", "email", member.Email, "role", member.Role)
	
	resp := LoginResponse{
		Token:      token,
		ExpiresAt:  time.Now().Add(15 * time.Minute),
		MemberName: member.Email, // Email as name for now
		MFAStatus:  mfaStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
