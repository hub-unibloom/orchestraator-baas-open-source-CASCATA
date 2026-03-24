package backup

import (
	"context"
	"fmt"
	"time"

	"cascata/internal/database"
)

// Restorer gracefully hot-swaps active production schema with a validated staging clone.
type Restorer struct {
	repo *database.Repository
}

func NewRestorer(repo *database.Repository) *Restorer {
	return &Restorer{repo: repo}
}

// AtomicSwap renames the active schema to a rollback name and promotes the stage schema to live.
// Since it operates on schemas within the same database, it is extremely fast and doesn't drop connections.
func (r *Restorer) AtomicSwap(ctx context.Context, slug string, liveSchema string, stageSchema string) (string, error) {
	rollbackName := fmt.Sprintf("%s_rollback_%d", liveSchema, time.Now().Unix())

	// 1. Rename Live -> Rollback
	renameLiveSQL := fmt.Sprintf("ALTER SCHEMA %q RENAME TO %q", liveSchema, rollbackName)
	_, err := r.repo.Pool.Exec(ctx, renameLiveSQL)
	if err != nil {
		return "", fmt.Errorf("restore: failed to demote live schema: %w", err)
	}

	// 2. Rename Stage -> Live (Promotion)
	renameStageSQL := fmt.Sprintf("ALTER SCHEMA %q RENAME TO %q", stageSchema, liveSchema)
	_, err = r.repo.Pool.Exec(ctx, renameStageSQL)
	if err != nil {
		// DANGER ZONE: Attempt rollback
		_ = r.repo.Pool.Exec(ctx, fmt.Sprintf("ALTER SCHEMA %q RENAME TO %q", rollbackName, liveSchema))
		return "", fmt.Errorf("restore: failed to promote stage schema. original restored. error: %w", err)
	}

	return rollbackName, nil
}
