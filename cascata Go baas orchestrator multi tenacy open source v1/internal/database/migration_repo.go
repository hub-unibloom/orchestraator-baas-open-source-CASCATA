package database

import (
	"context"
	"fmt"

	"cascata/internal/domain"
	"github.com/jackc/pgx/v5"
)

// MigrationRepository handles the persistence of schema versioning within a tenant database.
type MigrationRepository struct {
	pool *Repository
}

func NewMigrationRepository(pool *Repository) *MigrationRepository {
	return &MigrationRepository{pool: pool}
}

// EnsureHistoryTable initializes the migration tracking metadata if not present.
func (r *MigrationRepository) EnsureHistoryTable(ctx context.Context) error {
	_, err := r.pool.Pool.Exec(ctx, domain.MigrationTableDDL)
	return err
}

// GetAppliedMigrations retrieves the list of successfully executed schema changes.
func (r *MigrationRepository) GetAppliedMigrations(ctx context.Context) ([]domain.Migration, error) {
	rows, err := r.pool.Pool.Query(ctx, "SELECT id, slug, name, ddl, checksum, status, applied_at FROM cascata_migrations ORDER BY applied_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []domain.Migration
	for rows.Next() {
		var m domain.Migration
		if err := rows.Scan(&m.ID, &m.Slug, &m.Name, &m.DDL, &m.Checksum, &m.Status, &m.AppliedAt); err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}
	return migrations, nil
}

// RecordStep logs the attempt or result of a structural change.
func (r *MigrationRepository) RecordStep(ctx context.Context, tx pgx.Tx, m *domain.Migration) error {
	sql := `
		INSERT INTO cascata_migrations (slug, name, ddl, checksum, status, error, applied_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (name) DO UPDATE SET
			status = EXCLUDED.status,
			error = EXCLUDED.error,
			applied_at = EXCLUDED.applied_at;
	`
	_, err := tx.Exec(ctx, sql, m.Slug, m.Name, m.DDL, m.Checksum, m.Status, m.Error, m.AppliedAt)
	return err
}

// AcquireLock uses Postgres Advisory Locks to ensure single-orchestrator execution (Phase 15).
func (r *MigrationRepository) AcquireLock(ctx context.Context, tx pgx.Tx, lockID int64) error {
	var acquired bool
	err := tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", lockID).Scan(&acquired)
	if err != nil {
		return err
	}
	if !acquired {
		return fmt.Errorf("database: migration lock already held by another process")
	}
	return nil
}
