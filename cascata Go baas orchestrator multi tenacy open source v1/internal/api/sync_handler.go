package api

import (
	"encoding/json"
	"net/http"

	"cascata/internal/domain"
	"cascata/internal/service"
	"github.com/go-chi/chi/v5"
)

// SyncHandler manages the convergent data synchronization endpoints.
type SyncHandler struct {
	syncSvc *service.SyncService
}

func NewSyncHandler(syncSvc *service.SyncService) *SyncHandler {
	return &SyncHandler{syncSvc: syncSvc}
}

// HandleSync processes a batch of CRDT operations for a specific table.
func (h *SyncHandler) HandleSync(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "project")
	table := chi.URLParam(r, "table")

	var payload domain.SyncPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid sync payload")
		return
	}

	payload.ProjectSlug = projectSlug
	payload.Table = table

	results, err := h.syncSvc.MergeBatch(r.Context(), &payload)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "Sync failure", err.Error())
		return
	}

	SendJSON(w, r, http.StatusOK, results)
}
