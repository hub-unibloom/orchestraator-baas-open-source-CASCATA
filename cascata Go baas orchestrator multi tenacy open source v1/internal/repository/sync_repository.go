package repository

import (
	"context"
	"fmt"
	"time"

	"cascata/internal/domain"
	"github.com/jackc/pgx/v5"
)

// SyncRepository handles the storage logic for record convergence (CRDT-LWW).
// It abstracts the atomic upsert with cascata-meta-tracking.
type SyncRepository struct{}

func NewSyncRepository() *SyncRepository {
	return &SyncRepository{}
}

// GetRecordMetadata retrieves only the sync-related metadata for a record.
func (r *SyncRepository) GetRecordMetadata(ctx context.Context, tx pgx.Tx, table string, id string) (time.Time, bool, error) {
	var updatedAt time.Time
	sql := fmt.Sprintf("SELECT cascata_updated_at FROM %s WHERE id = $1 FOR UPDATE", table)
	err := tx.QueryRow(ctx, sql, id).Scan(&updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows { return time.Time{}, false, nil }
		return time.Time{}, false, err
	}
	return updatedAt, true, nil
}

// UpsertRecord applies the convergent change into the tenant table.
// Supports dynamic payloads (JSONB or Columns mapping) with Sync Meta-Columns.
func (r *SyncRepository) UpsertRecord(ctx context.Context, tx pgx.Tx, table string, op domain.SyncOperation) error {
	var deletedAt interface{} = nil
	if op.IsDeleted {
		deletedAt = op.UpdatedAt
	}

	// For Phase 11, we assume 'data' is a JSONB column or we map it.
	// Production-Grade: If 'data' is present in Op, we use it. 
	// If the table is structured, we'd need a more complex mapping.
	// For simplicity in the generic engine, we treat 'data' as the main bucket.
	sql := fmt.Sprintf(`
		INSERT INTO %s (id, data, cascata_updated_at, cascata_deleted_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			data = EXCLUDED.data,
			cascata_updated_at = EXCLUDED.cascata_updated_at,
			cascata_deleted_at = EXCLUDED.cascata_deleted_at
		WHERE %s.cascata_updated_at < EXCLUDED.cascata_updated_at;
	`, table, table)

	_, err := tx.Exec(ctx, sql, op.ID, op.Data, op.UpdatedAt, deletedAt)
	return err
}
