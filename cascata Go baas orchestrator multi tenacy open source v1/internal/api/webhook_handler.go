package api

import (
	"log/slog"
	"net/http"

	"cascata/internal/webhook"
	"github.com/go-chi/chi/v5"
)

// WebhookHandler handles inbound triggers from external services.
type WebhookHandler struct {
	svc *webhook.Service
}

func NewWebhookHandler(svc *webhook.Service) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

// ServeHTTP processes the inbound webhook request under the project scope.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "project")
	pathSlug := chi.URLParam(r, "path")

	slog.Info("webhook: incoming request", "project", projectSlug, "path", pathSlug)

	// Call the service for logic processing (HMAC, SSRF, Dispatch)
	status, msg := h.svc.HandleInbound(r.Context(), projectSlug, pathSlug, r)
	
	if status >= 400 {
		SendError(w, r, status, ErrUnauthorized, msg)
		return
	}

	SendJSON(w, r, status, map[string]string{"message": msg})
}
