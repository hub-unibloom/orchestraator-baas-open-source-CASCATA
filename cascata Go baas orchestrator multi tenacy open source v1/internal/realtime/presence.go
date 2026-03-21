package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cascata/internal/database"
)

// PresenceEntry represents a user's location in a channel.
type PresenceEntry struct {
	UserID     string                 `json:"user_id"`
	Channel    string                 `json:"channel"`
	Project    string                 `json:"project"`
	Metadata   map[string]interface{} `json:"metadata"`
	LastSeenAt time.Time              `json:"last_seen_at"`
}

// PresenceManager handles presence heartbeats and synchronization.
type PresenceManager struct {
	localPresence map[string]*PresenceEntry // userID -> entry
	mu            sync.RWMutex
	repo          *database.Repository
	manager       *Manager
}

func NewPresenceManager(repo *database.Repository, manager *Manager) *PresenceManager {
	return &PresenceManager{
		localPresence: make(map[string]*PresenceEntry),
		repo:          repo,
		manager:       manager,
	}
}

// Heartbeat updates our local state and notifies others.
func (p *PresenceManager) Heartbeat(ctx context.Context, userID, project, channel string, metadata map[string]interface{}) {
	entry := &PresenceEntry{
		UserID:     userID,
		Channel:    channel,
		Project:    project,
		Metadata:   metadata,
		LastSeenAt: time.Now(),
	}

	p.mu.Lock()
	p.localPresence[userID] = entry
	p.mu.Unlock()

	// 2. Broadcast Presence Change via SSE
	p.manager.Broadcast(project, Event{
		Table:    "presence",
		Action:   "heartbeat",
		Record:   entry,
		RecordID: userID,
	})
	
	// 3. Sync with Dragonfly (Inter-instance)
}

// RunBackgroundCleaner purges users who haven't sent a heartbeat for > 60s.
func (p *PresenceManager) RunBackgroundCleaner(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			now := time.Now()
			for id, entry := range p.localPresence {
				if now.Sub(entry.LastSeenAt) > 60*time.Second {
					delete(p.localPresence, id)
					// Broadcast Offline Status
					p.manager.Broadcast(entry.Project, Event{
						Table:    "presence",
						Action:   "offline",
						RecordID: id,
					})
				}
			}
			p.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
