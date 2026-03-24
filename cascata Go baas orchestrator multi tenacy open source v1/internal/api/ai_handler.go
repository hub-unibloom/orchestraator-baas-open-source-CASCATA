package api

import (
	"encoding/json"
	"net/http"

	"cascata/internal/ai"
	"cascata/internal/domain"
)

// AIHandler exposes the Orchestrator's intelligence layer as a BaaS (Phase 28).
type AIHandler struct {
	aiEngine *ai.Engine
}

func NewAIHandler(engine *ai.Engine) *AIHandler {
	return &AIHandler{aiEngine: engine}
}

// ServeAsk processes a user's prompt by injecting real-time tenant context.
func (h *AIHandler) ServeAsk(w http.ResponseWriter, r *http.Request) {
	authCtx, _ := domain.FromContext(r.Context())
	
	var config ai.PromptConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid AI prompt configuration")
		return
	}

	if config.UserPrompt == "" {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "User prompt is mandatory")
		return
	}

	response, err := h.aiEngine.ExecuteWithContext(r.Context(), authCtx.ProjectSlug, config)
	if err != nil {
		SendError(w, r, http.StatusBadGateway, ErrInternalError, "AI engine failed to respond", err.Error())
		return
	}

	SendJSON(w, r, http.StatusOK, map[string]string{
		"response": response,
		"model":    config.Model,
		"intent":   config.Intent,
	})
}
