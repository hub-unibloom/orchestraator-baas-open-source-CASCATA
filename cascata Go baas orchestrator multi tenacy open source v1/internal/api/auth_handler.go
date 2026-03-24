package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"cascata/internal/auth"
	"cascata/internal/domain"
	"cascata/internal/service"
	"github.com/go-chi/chi/v5"
)

// AuthHandler manages the identity lifecycle for Tenant Residents (end-users).
// Implements the Phase 10.1 strategies (CPF, WhatsApp, Magic Link).
type AuthHandler struct {
	residentAuth *auth.ResidentAuthService
	externalAuth *auth.ExternalAuthService
	projectSvc   *service.ProjectService
}

func NewAuthHandler(resAuth *auth.ResidentAuthService, extAuth *auth.ExternalAuthService, projectSvc *service.ProjectService) *AuthHandler {
	return &AuthHandler{
		residentAuth: resAuth,
		externalAuth: extAuth,
		projectSvc:   projectSvc,
	}
}

// HandleLogin processes authentication requests (Initial step for OTP or direct Password login).
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "project")
	
	var payload domain.AuthPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid auth payload")
		return
	}

	switch payload.Strategy {
	case domain.StrategyPassword:
		res, signedJWT, err := h.residentAuth.AuthenticateByCPF(r.Context(), projectSlug, payload.Identifier, payload.Password)
		if err != nil {
			slog.Warn("auth.resident: login lookup failed", "tenant_slug", projectSlug, "err", err)
			SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "AUTHENTICATION_FAILED")
			return
		}
		SendJSON(w, r, http.StatusOK, map[string]interface{}{
			"resident": res,
			"token":    signedJWT,
		})

	case domain.StrategyOTPWhatsApp:
		// Phase 10.1: Issues OTP to Dragonfly. Response code will be used in /auth/verify
		code, err := h.residentAuth.RequestWhatsAppOTP(r.Context(), projectSlug, payload.Identifier)
		if err != nil {
			SendError(w, r, http.StatusInternalServerError, ErrInternalError, "OTP_GENERATION_FAILURE")
			return
		}
		
		// In Phase 10.3 this will be sent via SMS/WhatsApp API. 
		// For now, we return 202 and log the code for testing.
		slog.Info("resident.auth: otp issued", "tenant_slug", projectSlug, "code", code)
		SendJSON(w, r, http.StatusAccepted, map[string]string{"message": "OTP_SENT_SUCCESSFULLY"})

	case domain.StrategyMagicLink:
		code, err := h.residentAuth.RequestMagicLink(r.Context(), projectSlug, payload.Identifier)
		if err != nil {
			SendError(w, r, http.StatusInternalServerError, ErrInternalError, "MAGIC_LINK_FAILURE")
			return
		}
		slog.Info("resident.auth: magic link code issued", "tenant_slug", projectSlug, "token", code)
		SendJSON(w, r, http.StatusAccepted, map[string]string{"message": "MAGIC_LINK_SENT_SUCCESSFULLY"})

	default:
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "UNSUPPORTED_STRATEGY")
	}
}

// HandleVerifyOTP completes the login flow for OTP-based strategies.
func (h *AuthHandler) HandleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "project")
	
	var payload struct {
		Identifier string `json:"identifier"`
		Code       string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid verify payload")
		return
	}

	res, signedJWT, err := h.residentAuth.VerifyOTP(r.Context(), projectSlug, payload.Identifier, payload.Code)
	if err != nil {
		SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "INVALID_OR_EXPIRED_CODE")
		return
	}

	SendJSON(w, r, http.StatusOK, map[string]interface{}{
		"resident": res,
		"token":    signedJWT,
	})
}

// HandleSignup creates a new Resident record in the project's auth.users table.
func (h *AuthHandler) HandleSignup(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "project")
	
	var res domain.Resident
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid resident details")
		return
	}

	// Capture raw password from metadata before it's processed
	rawPassword, _ := res.Metadata["password"].(string)
	delete(res.Metadata, "password") 

	err := h.residentAuth.SignupResident(r.Context(), projectSlug, &res, rawPassword)
	if err != nil {
		slog.Error("auth.resident: signup failed", "tenant_slug", projectSlug, "err", err)
		SendError(w, r, http.StatusConflict, ErrInvalidRequest, "RESIDENT_ALREADY_EXISTS_OR_INVALID_DATA")
		return
	}

	SendJSON(w, r, http.StatusCreated, res)
}

// HandleLogout invalidates the current session token (Phase 10.1).
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Token revocation is handled by Nginx/Dragonfly Blacklist in Tier 2
	SendJSON(w, r, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// HandleStepUpChallenge triggers an OTP for a user already authenticated.
func (h *AuthHandler) HandleStepUpChallenge(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := domain.FromContext(r.Context())
	if !ok || authCtx.Mode != domain.ModeResident {
		SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "Resident session required")
		return
	}

	err := h.residentAuth.IssueStepUpOTP(r.Context(), authCtx.ProjectSlug, authCtx.UserID)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "Challenge initiation failed", err.Error())
		return
	}

	SendJSON(w, r, http.StatusOK, map[string]string{"message": "Step-up challenge initiated"})
}

// HandleVerifyStepUp validates the OTP and issues an elevated JWT.
func (h *AuthHandler) HandleVerifyStepUp(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := domain.FromContext(r.Context())
	if !ok || authCtx.Mode != domain.ModeResident {
		SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "Resident session required for step-up")
		return
	}
	
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid verification payload")
		return
	}

	token, err := h.residentAuth.VerifyStepUp(r.Context(), authCtx.ProjectSlug, authCtx.UserID, payload.Code)
	if err != nil {
		SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "Verification failed", err.Error())
		return
	}

	SendJSON(w, r, http.StatusOK, map[string]string{"access_token": token, "token_type": "Bearer"})
}

// HandleAuthorize initiates the OAuth redirect for a third-party provider.
func (h *AuthHandler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "project")
	provider := r.URL.Query().Get("provider")
	redirectTo := r.URL.Query().Get("redirect_to")

	if provider == "" {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Provider is required")
		return
	}

	url, err := h.externalAuth.AuthorizeProvider(r.Context(), projectSlug, provider, redirectTo)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "Authorize initiation failed", err.Error())
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandleCallback processes the external provider redirection.
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "project")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	res, token, err := h.externalAuth.Callback(r.Context(), projectSlug, "google", code, state)
	if err != nil {
		SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "External auth callback failed", err.Error())
		return
	}

	// Redirect back to resident application with session token
	redirectURL := fmt.Sprintf("http://localhost:3000?token=%s&resident=%s", token, res.ID)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
