package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"cascata/internal/auth"
	"cascata/internal/crypto"
	"cascata/internal/domain"
	"cascata/internal/i18n"
	"cascata/internal/ratelimit"
	"cascata/internal/repository"
	"cascata/internal/service"
	"cascata/internal/storage"
	"cascata/internal/ui/components"
	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"strconv"
	"strings"
)

// SystemHandler handles management-level authentication and sessions.
// Orchestrates the "Front-Door" of the Cascata Dashboard.
type SystemHandler struct {
	authSvc     *auth.SystemAuthService
	sessionSvc  *auth.SessionManager
	ratelimit   *ratelimit.AdaptiveEngine
	genesisSvc  *service.GenesisService
	projectSvc  *service.ProjectService
	ProjectRepo *repository.ProjectRepository // Keeping for low-level if needed, but projectSvc is preferred
	backupSvc   *storage.BackupService
}

func NewSystemHandler(
	authSvc *auth.SystemAuthService, 
	sessionSvc *auth.SessionManager, 
	rl *ratelimit.AdaptiveEngine,
	projectSvc *service.ProjectService,
	projectRepo *repository.ProjectRepository,
	backupSvc *storage.BackupService,
) *SystemHandler {
	return &SystemHandler{
		authSvc:     authSvc,
		sessionSvc:  sessionSvc,
		ratelimit:   rl,
		genesisSvc:  genesis,
		projectSvc:  projectSvc,
		ProjectRepo: projectRepo,
		backupSvc:   backupSvc,
	}
}

// REST Definitions for the System Context

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

// CreateProjectRequest defines the fields required to birth a new tenant.
type CreateProjectRequest struct {
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Region          string `json:"region"`
	TimeZone        string `json:"timezone"`
	MaxUsers        int    `json:"max_users"`
	MaxConns        int    `json:"max_conns"`
	MaxStorageMB    int64  `json:"max_storage_mb"`
	MaxDBWeightMB   int64  `json:"max_db_weight_mb"`
	MaxDataRows     int64  `json:"max_data_rows"`
	SecondarySecret string `json:"secondary_secret,omitempty"`
}

// HandleLogin processes the Worner/Member authentication flow.
func (h *SystemHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, r, http.StatusMethodNotAllowed, ErrInvalidRequest, "Method not allowed")
		return
	}

	var req LoginRequest
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid form data")
			return
		}
		req.Email = r.FormValue("email")
		req.Password = r.FormValue("password")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
			return
		}
	}

	// 1. Pre-auth Firewall (Lockout Check)
	if locked, ttl := h.ratelimit.IsLoginLocked(r.Context(), req.Email); locked {
		detail := fmt.Sprintf("Too many failed attempts. Try again in %s", ttl.Round(time.Second).String())
		SendError(w, r, http.StatusLocked, ErrRateLimit, "ACCOUNT_LOCKED", detail)
		return
	}

	// 2. Core Authentication (Password / Bcrypt)
	member, err := h.authSvc.AuthenticateMember(r.Context(), req.Email, req.Password)
	if err != nil {
		_ = h.ratelimit.RecordLoginFailure(r.Context(), req.Email)
		slog.Warn("system.login: authentication failed", "email", req.Email, "error", err)
		SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "INVALID_CREDENTIALS")
		return
	}

	// 3. Issue Session (15 Minutes Ephemeral as per Phase 3 Philosophy)
	token, err := h.sessionSvc.IssueSessionToken(r.Context(), member)
	if err != nil {
		slog.Error("system.login: session issuance failed", "error", err)
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "SESSION_FAILURE")
		return
	}

	// 4. Persistence (Browser Sovereignty)
	http.SetCookie(w, &http.Cookie{
		Name:     "cascata_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in prod with TLS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   900, // 15 minutes
	})

	slog.Info("system.login: session established", "email", member.Email, "role", member.Role)
	
	// 5. Response Strategy (HX-Sinergy)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}

	resp := LoginResponse{
		Token:      token,
		ExpiresAt:  time.Now().Add(15 * time.Minute),
		MemberName: member.Email,
		MFAStatus:  "passed",
	}
	SendJSON(w, r, http.StatusOK, resp)
}

// HandleCreateProject birthes a new tenant via the Genesis service.
func (h *SystemHandler) HandleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	
	// HTMX Synergy: Support for Form Data
	if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		_ = r.ParseForm()
		req.Name = r.FormValue("name")
		req.Slug = r.FormValue("slug")
		req.Region = "SOVEREIGN_NODE" // Default for Phase 1
		req.MaxUsers, _ = strconv.Atoi(r.FormValue("max_users"))
		req.MaxStorageMB, _ = strconv.ParseInt(r.FormValue("max_storage_mb"), 10, 64)
		req.MaxDataRows, _ = strconv.ParseInt(r.FormValue("max_data_rows"), 10, 64)
	} else {
		// Standard JSON API support
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid payload")
			return
		}
	}

	// 1. Prepare Metadata for the newborn project
	anon, svc, jwt, err := crypto.GenerateProjectCredentials()
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "CRYPTO_FAILURE")
		return
	}

	p := &domain.Project{
		Name:                req.Name,
		Slug:                req.Slug,
		DBName:              "cascata_meta", // Unified High-Density Pool
		Status:              "active",
		Region:              req.Region,
		TimeZone:            req.TimeZone,
		MaxUsers:            req.MaxUsers, // Concurrent Users
		MaxConns:            req.MaxConns,
		MaxStorageMB:        req.MaxStorageMB,
		MaxDBWeightMB:       req.MaxDBWeightMB,
		MaxDataRows:         req.MaxDataRows,
		SecondarySecretHash: req.SecondarySecret, // Hashing happens in service
		Metadata:            map[string]interface{}{"created_by": "worner_dashboard"},
		LogRetentionDays:    30,
		
		// Real Cryptographic Identities
		AnonKey:    anon,
		ServiceKey: svc,
		JWTSecret:  jwt,
	}

	// 2. Genesis Execution
	if err := h.genesisSvc.CreateProject(r.Context(), p); err != nil {
		slog.Error("system: failed to birth project", "slug", req.Slug, "err", err)
		SendError(w, r, http.StatusInternalServerError, ErrGenesisFailure, "GENESIS_FAILED", err.Error())
		return
	}

	// 3. System Auditor Ledger (Governance Sinergy)
	_ = h.authSvc.LogAudit(r.Context(), p.Slug, "PROJECT_BIRTH", "SYSTEM", "MEMBER", map[string]interface{}{
		"name":          p.Name,
		"max_users":     p.MaxUsers,
		"max_db_weight": p.MaxDBWeightMB,
	})

	// 4. Fragment Synergy (HTMX Response)
	if r.Header.Get("HX-Request") == "true" {
		loc := i18n.GetLocalizer(r)
		uiProject := components.Project{
			ID:           p.ID,
			Name:         p.Name,
			Slug:         p.Slug,
			Region:       p.Region,
			Status:       p.Status,
			MaxUsers:     p.MaxUsers,
			MaxConns:     p.MaxConns,
			MaxStorageMB: p.MaxStorageMB,
			MaxDataRows:  p.MaxDataRows,
		}
		templ.Handler(components.ProjectCard(uiProject, loc)).ServeHTTP(w, r)
		return
	}

	SendJSON(w, r, http.StatusCreated, p)
}

// HandleListProjects returns all projects currently managed by the Orquestrador.
func (h *SystemHandler) HandleListProjects(w http.ResponseWriter, r *http.Request) {
	// Sovereign Discovery: Use Service layer to ensure cascata_system context.
	projects, err := h.projectSvc.ListProjects(r.Context())
	if err != nil {
		slog.Error("system: project listing failed", "err", err)
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "LIST_FAILED", err.Error())
		return
	}

	SendJSON(w, r, http.StatusOK, projects)
}

// HandleDeleteProject removes a tenant from the orchestrator.
func (h *SystemHandler) HandleDeleteProject(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Missing slug")
		return
	}

	if err := h.genesisSvc.DeleteProject(r.Context(), slug); err != nil {
		slog.Error("system: failed to delete project", "slug", slug, "err", err)
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "DELETION_FAILED", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleExportCAF exports a project to a downloadable .caf file.
func (h *SystemHandler) HandleExportCAF(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Missing project slug")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.caf\"", slug))

	if err := h.backupSvc.ExportProject(r.Context(), slug, w); err != nil {
		slog.Error("system.backup: export failed", "slug", slug, "err", err)
		// Header might have been sent, but we log the error
	}
}
