package domain

import (
	"time"
)

// SyncPayload represents the standard package for offline-to-online synchronization.
// Implements the LWW-Element-Set (Last-Write-Wins) CRDT strategy.
type SyncPayload struct {
	Table       string                 `json:"table"`
	ProjectSlug string                 `json:"project_slug"`
	Operations  []SyncOperation        `json:"operations"`
}

// SyncOperation defines a single record change within a sync batch.
type SyncOperation struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	UpdatedAt time.Time              `json:"updated_at"`
	IsDeleted bool                   `json:"is_deleted,omitempty"`
}

// SyncResult confirms the resolution status for each record in the batch.
type SyncResult struct {
	ID        string `json:"id"`
	Status    string `json:"status"` // "merged", "rejected_older", "conflict_resolved"
	ServerTime time.Time `json:"server_time"`
}
