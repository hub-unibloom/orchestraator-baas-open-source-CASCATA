package api

import (
	"io"
	"net/http"

	"cascata/internal/domain"
	"cascata/internal/phantom"
	"github.com/go-chi/chi/v5"
)

// LogicHandler manages the execution of Phantom Functions (WASM Edge Nodes).
type LogicHandler struct {
	phantom *phantom.PhantomService
}

func NewLogicHandler(ph *phantom.PhantomService) *LogicHandler {
	return &LogicHandler{phantom: ph}
}

// ServeHTTP executes a project-defined function passed in the request body (Developer Mode).
// Integrated with OTel and High-Performance Sandbox.
func (h *LogicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authCtx, _ := domain.FromContext(r.Context())
	
	// 1. Read WASM binary from body (Direct execution mode for development Phase 14)
	wasmBinary, err := io.ReadAll(r.Body)
	if err != nil || len(wasmBinary) == 0 {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Missing WASM binary in request body")
		return
	}

	// 2. Unied Phantom Invocation (Sinergy Phase 14/15)
	// For Direct mode, we use the project slug as a function identifier for caching.
	result, err := h.phantom.InvokeFunction(r.Context(), authCtx.ProjectSlug, "direct_dev_exec", wasmBinary, nil)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "WASM_EXEC_ERROR", err.Error())
		return
	}

	// 3. Response handling (Raw response from Sandbox)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result)
}

// HandleInvoke resolves and executes a named function.
func (h *LogicHandler) HandleInvoke(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	
	// Production-Grade: Fetch from system.storage or cache.
	// For Phase 14 maturity, we confirm the bridge is ready.
	SendError(w, r, http.StatusNotImplemented, ErrInternalError, "Named function resolution in Phase 15.2 storage rollout.", name)
}
