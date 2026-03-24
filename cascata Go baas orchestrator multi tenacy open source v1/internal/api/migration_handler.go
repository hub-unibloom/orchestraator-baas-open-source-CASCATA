package api

import (
	"encoding/json"
	"net/http"

	"cascata/internal/domain"
	"cascata/internal/service"
)

// MigrationHandler manages schema evolution requests for tenants (Phase 15).
type MigrationHandler struct {
	migrationSvc *service.MigrationService
}

func NewMigrationHandler(svc *service.MigrationService) *MigrationHandler {
	return &MigrationHandler{migrationSvc: svc}
}

// ServeMigration processes a single structural DDL command (Zero-Downtime Engine).
// Integrated with OpenTelemetry and Audit Service via MigrationService.
func (h *MigrationHandler) ServeMigration(w http.ResponseWriter, r *http.Request) {
	authCtx, _ := domain.FromContext(r.Context())
	
	var payload struct {
		Name string `json:"name"`
		DDL  string `json:"ddl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid migration payload", err.Error())
		return
	}

	if payload.Name == "" || payload.DDL == "" {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Migration 'name' and 'ddl' are mandatory")
		return
	}

	err := h.migrationSvc.ApplyMigration(r.Context(), authCtx.ProjectSlug, payload.Name, payload.DDL)
	if err != nil {
		SendError(w, r, http.StatusConflict, ErrInternalError, "Migration attempt failed", err.Error())
		return
	}

	SendJSON(w, r, http.StatusOK, map[string]interface{}{
		"message":  "Migration applied successfully",
		"name":     payload.Name,
		"project":  authCtx.ProjectSlug,
	})
}

// GetMigrationHistory retrieves the list of all schema changes for the current tenant.
func (h *MigrationHandler) GetMigrationHistory(w http.ResponseWriter, r *http.Request) {
	authCtx, _ := domain.FromContext(r.Context())
	
	// TODO: Add history retrieval to MigrationService (Phase 15.2 expansion)
	SendJSON(w, r, http.StatusOK, map[string]interface{}{
		"project": authCtx.ProjectSlug,
		"history": []string{},
	})
}
