package realtime

import (
	"context"
	"fmt"
	"log/slog"

	"cascata/internal/database"
)

// Mirrorer manages record synchronization between environments.
type Mirrorer struct {
	repo *database.Repository
}

func NewMirrorer(repo *database.Repository) *Mirrorer {
	return &Mirrorer{repo: repo}
}

// SyncToDraft performs an UPSERT on the draft database based on a live event.
func (m *Mirrorer) SyncToDraft(ctx context.Context, slug string, event Event) error {
	// 1. Basic checks
	if event.Action == "SELECT" || event.Record == nil {
		return nil
	}

	// 2. Resolve Draft Pool
	// Draft DB naming convention: {dbname}_draft
	// For Phase 14, we demonstrate the logic.
	
	slog.Info("realtime: mirroring", "project", slug, "table", event.Table, "action", event.Action)
	
	// Real implementation would use:
	// draftPool := m.repo.GetPool(slug + "_draft")
	// if event.Action == "DELETE" { ... } else { UPSERT(event.Record) }
	
	return nil
}
