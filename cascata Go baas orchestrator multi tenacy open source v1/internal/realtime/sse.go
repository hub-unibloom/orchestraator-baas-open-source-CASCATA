package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"cascata/internal/database"
)

// Event represents a single data change event.
type Event struct {
	Table     string      `json:"table"`
	Action    string      `json:"action"`
	Record    interface{} `json:"record,omitempty"`
	RecordID  interface{} `json:"record_id"`
	Timestamp string      `json:"timestamp"`
}

// Client represents a single SSE connection.
type Client struct {
	ID          string
	ProjectSlug string
	Send        chan string
	Done        <-chan struct{}
}

// Manager orchestrates thousands of concurrent SSE connections.
type Manager struct {
	subscribers map[string]map[string]*Client // slug -> clientID -> client
	mu          sync.RWMutex
	repo        *database.Repository
}

func NewManager(repo *database.Repository) *Manager {
	return &Manager{
		subscribers: make(map[string]map[string]*Client),
		repo:        repo,
	}
}

// HandleSSE manages the lifecycle of an SSE connection.
func (m *Manager) HandleSSE(w http.ResponseWriter, r *http.Request, slug string) {
	// 1. Headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	clientID := fmt.Sprintf("%d", time.Now().UnixNano())
	client := &Client{
		ID:          clientID,
		ProjectSlug: slug,
		Send:        make(chan string, 100), // Buffer to handle small bursts
		Done:        r.Context().Done(),
	}

	// 2. Register Client
	m.mu.Lock()
	if m.subscribers[slug] == nil {
		m.subscribers[slug] = make(map[string]*Client)
	}
	m.subscribers[slug][clientID] = client
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.subscribers[slug], clientID)
		if len(m.subscribers[slug]) == 0 {
			delete(m.subscribers, slug)
		}
		m.mu.Unlock()
	}()

	// 3. Keep-alive Heartbeat
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Initial message
	fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected"}`)
	flusher.Flush()

	for {
		select {
		case msg := <-client.Send:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case <-client.Done:
			return
		}
	}
}

// Broadcast sends an event to all clients in a project.
func (m *Manager) Broadcast(slug string, event Event) {
	m.mu.RLock()
	clients, ok := m.subscribers[slug]
	m.mu.RUnlock()

	if !ok {
		return
	}

	msg, _ := json.Marshal(event)
	msgStr := string(msg)

	for _, client := range clients {
		select {
		case client.Send <- msgStr:
		default:
			// Circuit Breaker: Slow consumer gets skipped or disconnected
			// For now, we just skip to avoid blocking the broadcaster.
		}
	}
}
