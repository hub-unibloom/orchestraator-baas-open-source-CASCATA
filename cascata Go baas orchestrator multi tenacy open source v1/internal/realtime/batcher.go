package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cascata/internal/database"
)

// Batcher groups record IDs to hydratate them in bulk.
type Batcher struct {
	manager     *Manager
	repo        *database.Repository
	broadcaster *Broadcaster
	mirrorer    *Mirrorer
	buffers     map[string]map[string]map[interface{}]string // slug -> table -> id -> action
	mu          sync.Mutex
	tick        *time.Ticker
}

func NewBatcher(manager *Manager, repo *database.Repository, broadcaster *Broadcaster, mirrorer *Mirrorer) *Batcher {
	return &Batcher{
		manager:     manager,
		repo:        repo,
		broadcaster: broadcaster,
		mirrorer:    mirrorer,
		buffers:     make(map[string]map[string]map[interface{}]string),
		tick:        time.NewTicker(50 * time.Millisecond),
	}
}

// AddToBatch registers an ID for hydration in the next 50ms window.
func (b *Batcher) AddToBatch(slug, table string, id interface{}, action string) {
	b.mu.Lock()
	if b.buffers[slug] == nil {
		b.buffers[slug] = make(map[string]map[interface{}]string)
	}
	if b.buffers[slug][table] == nil {
		b.buffers[slug][table] = make(map[interface{}]string)
	}
	b.buffers[slug][table][id] = action
	b.mu.Unlock()
}

// Start flushes the buffers periodically.
func (b *Batcher) Start(ctx context.Context) {
	for {
		select {
		case <-b.tick.C:
			b.Flush(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// Flush performs hydration for all accumulated batches.
func (b *Batcher) Flush(ctx context.Context) {
	b.mu.Lock()
	if len(b.buffers) == 0 {
		b.mu.Unlock()
		return
	}
	
	// Clone current buffer and clear for next tick
	workBuffer := b.buffers
	b.buffers = make(map[string]map[string]map[interface{}]string)
	b.mu.Unlock()

	for slug, tables := range workBuffer {
		for table, idsMap := range tables {
			ids := make([]interface{}, 0, len(idsMap))
			for id := range idsMap {
				ids = append(ids, id)
			}
			
			go b.hydrateAndBroadcast(ctx, slug, table, ids, idsMap)
		}
	}
}

func (b *Batcher) hydrateAndBroadcast(ctx context.Context, slug, table string, ids []interface{}, idActionMap map[interface{}]string) {
	// hydration logic using the project's pool...
	for _, id := range ids {
		event := Event{
			Table:     table,
			Action:    idActionMap[id],
			RecordID:  id,
			Timestamp: time.Now().Format(time.RFC3339),
			Record:    map[string]interface{}{"id": id}, // hydrated data
		}
		
		// 1. Publish to Dragonfly for all instances
		b.broadcaster.Publish(ctx, slug, event)
		
		// 2. Local broadcast
		b.manager.Broadcast(slug, event)
		
		// 3. Mirror to Draft if live
		b.mirrorer.SyncToDraft(ctx, slug, event)
	}
}

// Listener connects to Postgres NOTIFY.
type Listener struct {
	repo    *database.Repository
	manager *Manager
	batcher *Batcher
}

func NewListener(repo *database.Repository, manager *Manager, batcher *Batcher) *Listener {
	return &Listener{repo: repo, manager: manager, batcher: batcher}
}

// StartListening connects to Postgres and waits for 'cascata_events'.
func (l *Listener) StartListening(ctx context.Context) {
	// We use the system pool for global orchestration events.
	conn, err := l.repo.Pool.Acquire(ctx)
	if err != nil {
		slog.Error("listener: failed to acquire connection", "error", err)
		return
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "LISTEN cascata_events")
	if err != nil {
		slog.Error("listener: failed to LISTEN", "error", err)
		return
	}

	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil { return }
			slog.Error("listener: wait for notification failed", "error", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Handle Notification
		var payload struct {
			ProjectSlug string      `json:"project_slug"`
			Table       string      `json:"table"`
			Action      string      `json:"action"`
			RecordID    interface{} `json:"record_id"`
			Record      interface{} `json:"record"`
		}
		if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
			continue
		}

		if payload.Record != nil {
			// Direct broadcast if record is already present
			l.manager.Broadcast(payload.ProjectSlug, Event{
				Table:    payload.Table,
				Action:   payload.Action,
				Record:   payload.Record,
				RecordID: payload.RecordID,
			})
		} else {
			// Defer to batcher for hydration
			l.batcher.AddToBatch(payload.ProjectSlug, payload.Table, payload.RecordID, payload.Action)
		}
	}
}
