package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/privacy"
	"cascata/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
)

type RealtimeHub struct {
	projectSvc    *service.ProjectService
	privacyEngine *privacy.Engine
	mu            sync.RWMutex
	listeners     map[string]*tenantBroadcaster
	
	// Unified Global Listener (Phase 22: High Density)
	masterRepo    *database.Repository
	stopOnce      sync.Once
}

func NewRealtimeHub(projectSvc *service.ProjectService, pEng *privacy.Engine, masterRepo *database.Repository) *RealtimeHub {
	h := &RealtimeHub{
		projectSvc:    projectSvc,
		privacyEngine: pEng,
		listeners:     make(map[string]*tenantBroadcaster),
		masterRepo:    masterRepo,
	}
	
	// Start the single global listener loop for the entire database heart.
	go h.globalListenLoop()
	
	return h
}

// tenantBroadcaster manages SSE clients for a specific tenant.
type tenantBroadcaster struct {
	slug    string
	clients map[chan []byte]*domain.AuthContext
	mu      sync.RWMutex
}

// HandleSSE establishes a persistent high-frequency connection for a project.
func (h *RealtimeHub) HandleSSE(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := domain.FromContext(r.Context())
	if !ok {
		SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "Authentication required")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	b := h.getOrCreateBroadcaster(authCtx.ProjectSlug)

	clientChan := make(chan []byte, 100)
	b.mu.Lock()
	b.clients[clientChan] = authCtx
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.clients, clientChan)
		b.mu.Unlock()
		close(clientChan)
	}()

	fmt.Fprintf(w, "event: system\ndata: {\"status\": \"bound\", \"slug\": %q}\n\n", b.slug)
	w.(http.Flusher).Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", string(msg))
			w.(http.Flusher).Flush()
		case <-time.After(30 * time.Second):
			fmt.Fprintf(w, ": heartbeat\n\n")
			w.(http.Flusher).Flush()
		}
	}
}

func (h *RealtimeHub) getOrCreateBroadcaster(slug string) *tenantBroadcaster {
	h.mu.Lock()
	defer h.mu.Unlock()

	if b, ok := h.listeners[slug]; ok {
		return b
	}

	b := &tenantBroadcaster{
		slug:    slug,
		clients: make(map[chan []byte]*domain.AuthContext),
	}
	h.listeners[slug] = b
	return b
}

// globalListenLoop sits on a single Postgres connection and routes all notifications.
func (h *RealtimeHub) globalListenLoop() {
	ctx := context.Background()
	slog.Info("realtime: starting unified global listener")
	
	for {
		conn, err := h.masterRepo.Pool.Acquire(ctx)
		if err != nil {
			slog.Error("realtime: global acquire failed, retrying", "err", err)
			time.Sleep(5 * time.Second)
			continue
		}

		_, err = conn.Exec(ctx, "LISTEN cascata_realtime")
		if err != nil {
			conn.Release()
			slog.Error("realtime: global listen failed, retrying", "err", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for {
			notification, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				conn.Release()
				slog.Error("realtime: global wait failed, reconnecting", "err", err)
				break
			}

			// Route notification to the correct broadcaster based on 'schema' (tenant slug)
			h.routeNotification(notification)
		}
	}
}

func (h *RealtimeHub) routeNotification(notification *pgconn.Notification) {
	var payload struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
		slog.Warn("realtime: invalid payload received", "err", err)
		return
	}

	h.mu.RLock()
	b, ok := h.listeners[payload.Schema]
	h.mu.RUnlock()

	if ok {
		h.broadcast(b, notification)
	}
}

// DeltaPayload represents the refined change data packet.
type DeltaPayload struct {
	Table     string                 `json:"table"`
	Op        string                 `json:"op"`
	Data      map[string]interface{} `json:"data"`
	Delta     map[string]interface{} `json:"delta,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

func (h *RealtimeHub) broadcast(b *tenantBroadcaster, notification *pgconn.Notification) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.clients) == 0 {
		return
	}

	var raw struct {
		Table string                 `json:"table"`
		Op    string                 `json:"op"`
		Data  map[string]interface{} `json:"data"`
		Old   map[string]interface{} `json:"old"`
	}

	if err := json.Unmarshal([]byte(notification.Payload), &raw); err != nil {
		slog.Error("realtime: unmarshal failure", "slug", b.slug, "err", err)
		return
	}

	// 1. Resolve Project for Privacy Config (Phase 7 Integration)
	proj, err := h.projectSvc.Resolve(context.Background(), b.slug)
	if err != nil {
		slog.Error("realtime: project resolution failed during broadcast", "slug", b.slug, "err", err)
		return
	}

	var privacyCfg privacy.ProjectPrivacyConfig
	if rawPriv, ok := proj.Metadata["privacy_config"]; ok {
		rawB, _ := json.Marshal(rawPriv)
		_ = json.Unmarshal(rawB, &privacyCfg)
	}

	// 2. Performance: Group clients by Role to avoid redundant masking in hot path
	rolePayloads := make(map[string][]byte)

	for client, authCtx := range b.clients {
		payload, exists := rolePayloads[authCtx.Role]
		if !exists {
			// Calculate delta for UPDATE operations
			var delta map[string]interface{}
			if raw.Op == "UPDATE" && raw.Old != nil {
				delta = h.calculateDelta(raw.Old, raw.Data)
			}

			dp := DeltaPayload{
				Table:     raw.Table,
				Op:        raw.Op,
				Data:      raw.Data,
				Delta:     delta,
				Timestamp: time.Now(),
			}

			// Apply Privacy Masking (Phase 7)
			// ApplyMasking works on []map[string]any, so we wrap and unwrap.
			rows := []map[string]interface{}{dp.Data}
			maskedRows, _ := h.privacyEngine.ApplyMasking(context.Background(), authCtx, privacyCfg, raw.Table, rows)
			dp.Data = maskedRows[0]

			if dp.Delta != nil {
				deltaRows := []map[string]interface{}{dp.Delta}
				maskedDelta, _ := h.privacyEngine.ApplyMasking(context.Background(), authCtx, privacyCfg, raw.Table, deltaRows)
				dp.Delta = maskedDelta[0]
			}

			payload, _ = json.Marshal(dp)
			rolePayloads[authCtx.Role] = payload
		}

		// 3. Dispatch to client channel with non-blocking select (Antifragile delivery)
		select {
		case client <- payload:
		default:
			// Client channel full, skip to maintain cluster reactivity
		}
	}
}

// Broadcast sends a custom payload to all listeners of a specific project (Phase 13/14 Synergy).
// It acts as the direct bridge from Automation/Logic handlers to the Real-time stream.
func (h *RealtimeHub) Broadcast(ctx context.Context, slug string, payload interface{}) error {
	h.mu.RLock()
	b, ok := h.listeners[slug]
	h.mu.RUnlock()

	if !ok {
		return nil // No active listeners for this tenant
	}

	// 1. Prepare DeltaPayload wrapper
	dp := DeltaPayload{
		Table:     "automation_signal",
		Op:        "CUSTOM",
		Data:      map[string]interface{}{"payload": payload},
		Timestamp: time.Now(),
	}

	// 2. Wrap notification and broadcast (Privacy aware grouping is handled internally by broadcast)
	// Since broadcast expects a *pgconn.Notification, we simulate one.
	// REFINEMENT: we should refactor broadcast to take a generic payload or separate the logic.
	
	rawJson, _ := json.Marshal(map[string]interface{}{
		"table": "automation_signal",
		"op": "CUSTOM",
		"data": dp.Data,
	})
	
	h.broadcast(b, &pgconn.Notification{Payload: string(rawJson)})
	return nil
}

// calculateDelta extracts the changed fields between two maps.
func (h *RealtimeHub) calculateDelta(old, new map[string]interface{}) map[string]interface{} {
	delta := make(map[string]interface{})
	for k, v := range new {
		oldVal, exists := old[k]
		if !exists || fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", v) {
			delta[k] = v
		}
	}
	return delta
}
