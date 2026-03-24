package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"cascata/internal/domain"
	"cascata/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RealtimeHub struct {
	projectSvc    *service.ProjectService
	privacyEngine *privacy.Engine
	mu            sync.RWMutex
	listeners     map[string]*tenantBroadcaster
}

func NewRealtimeHub(projectSvc *service.ProjectService, pEng *privacy.Engine) *RealtimeHub {
	return &RealtimeHub{
		projectSvc:    projectSvc,
		privacyEngine: pEng,
		listeners:     make(map[string]*tenantBroadcaster),
	}
}

// tenantBroadcaster manages listeners for a specific physical database pool.
type tenantBroadcaster struct {
	slug    string
	ctx     context.Context
	cancel  context.CancelFunc
	clients map[chan []byte]*domain.AuthContext
	mu      sync.RWMutex
}

// HandleSSE establishes a persistent high-frequency connection for a project.
// GET /v1/{project}/realtime
func (h *RealtimeHub) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// 1. Authenticate and resolve identity
	authCtx, ok := domain.FromContext(r.Context())
	if !ok {
		SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "Authentication required for real-time stream")
		return
	}

	// 2. Prepare SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 3. Obtain or spawn a Broadcaster for this tenant
	b, err := h.getBroadcaster(r.Context(), authCtx.Project)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "REALTIME_ORCHESTRATION_FAILED", err.Error())
		return
	}

	// 4. Register client channel with AuthContext for role-based privacy masking (Phase 13 Sinergy)
	clientChan := make(chan []byte, 100)
	b.mu.Lock()
	b.clients[clientChan] = authCtx
	b.mu.Unlock()

	// 5. Cleanup on disconnect
	defer func() {
		b.mu.Lock()
		delete(b.clients, clientChan)
		b.mu.Unlock()
		close(clientChan)
		slog.Debug("realtime: client disconnected", "slug", b.slug)
	}()

	// 6. Signal Stream Initialization
	fmt.Fprintf(w, "event: system\ndata: {\"status\": \"bound\", \"slug\": %q}\n\n", b.slug)
	w.(http.Flusher).Flush()

	// 7. Pulse Handling Loop
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", string(msg))
			w.(http.Flusher).Flush()
		case <-time.After(30 * time.Second):
			// Keep-alive heartbeat
			fmt.Fprintf(w, ": heartbeat\n\n")
			w.(http.Flusher).Flush()
		}
	}
}

func (h *RealtimeHub) getBroadcaster(ctx context.Context, p *domain.Project) (*tenantBroadcaster, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if b, ok := h.listeners[p.Slug]; ok {
		return b, nil
	}

	// Spawn new listener for this tenant
	slog.Info("realtime: spawning broadcaster for tenant", "slug", p.Slug)
	
	poolRepo, err := h.projectSvc.GetPool(ctx, p)
	if err != nil {
		return nil, err
	}

	bCtx, cancel := context.WithCancel(context.Background())
	b := &tenantBroadcaster{
		slug:    p.Slug,
		ctx:     bCtx,
		cancel:  cancel,
		clients: make(map[chan []byte]*domain.AuthContext),
	}

	go h.listenLoop(b, poolRepo.Pool)
	
	h.listeners[p.Slug] = b
	return b, nil
}

// listenLoop sits on the Postgres LISTEN queue for a specific tenant DB.
func (h *RealtimeHub) listenLoop(b *tenantBroadcaster, pool *pgxpool.Pool) {
	slog.Info("realtime: listener loop started", "slug", b.slug)
	
	// We need a dedicated connection for LISTEN/NOTIFY.
	conn, err := pool.Acquire(b.ctx)
	if err != nil {
		slog.Error("realtime: failed to acquire listener connection", "slug", b.slug, "err", err)
		return
	}
	defer conn.Release()

	// Command the database to alert the orchestrator of any change in the 'public' schema or custom triggers.
	_, err = conn.Exec(b.ctx, "LISTEN cascata_realtime")
	if err != nil {
		slog.Error("realtime: failed to listen on channel", "slug", b.slug, "err", err)
		return
	}

	for {
		notification, err := conn.Conn().WaitForNotification(b.ctx)
		if err != nil {
			if !b.ctx.Err() != nil {
				slog.Error("realtime: notification wait error", "slug", b.slug, "err", err)
			}
			return
		}

		// Broadcast to all active SSE clients (Phase 13: Delta aware)
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
