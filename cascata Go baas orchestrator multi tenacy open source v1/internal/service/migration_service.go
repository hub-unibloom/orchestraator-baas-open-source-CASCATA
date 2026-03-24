package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"log/slog"
	"strings"
	"time"

	"cascata/internal/database"
	"cascata/internal/domain"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MigrationService is the "Structural Orchestrator" of Cascata (Phase 15).
// It applies schema changes to tenant pools with zero-downtime, collision detection and OTel tracing.
type MigrationService struct {
	projectSvc *ProjectService
	audit      *AuditService
}

func NewMigrationService(projectSvc *ProjectService, audit *AuditService) *MigrationService {
	return &MigrationService{
		projectSvc: projectSvc,
		audit:      audit,
	}
}

// ApplyMigration executes DDL safely, tracking results in a project's migration table.
// Implements transactional advisory locks and OTel instrumentation.
func (s *MigrationService) ApplyMigration(ctx context.Context, slug string, name string, ddl string) error {
	tr := otel.Tracer("migration-engine")
	ctx, span := tr.Start(ctx, fmt.Sprintf("migration: %s", name), trace.WithAttributes(
		attribute.String("cascata.project", slug),
		attribute.String("migration.name", name),
	))
	defer span.End()

	slog.Info("migration: starting structural change", "slug", slug, "name", name)

	// 1. Calculate and verify checksum
	checksum := s.calculateChecksum(ddl)
	
	// 2. Resolve Project & Acquire Pool
	p, err := s.projectSvc.Resolve(ctx, slug)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("migration.ApplyMigration: project resolution failed: %w", err)
	}

	pool, err := s.projectSvc.GetPool(ctx, p)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("migration.ApplyMigration: pool resolution failed: %w", err)
	}

	repo := database.NewMigrationRepository(pool)

	// 3. Ensure History Table Exists (Genesis Pattern)
	if err := repo.EnsureHistoryTable(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("migration.ApplyMigration: history table creation failed: %w", err)
	}

	// 4. Check for 'CONCURRENTLY' - these cannot be run inside a transaction
	isConcurrent := strings.Contains(strings.ToUpper(ddl), "CONCURRENTLY")

	// 5. Execution Workflow
	err = pgx.BeginFunc(ctx, pool.Pool, func(tx pgx.Tx) error {
		// a. Acquire Advisory Lock to prevent concurrent orchestrators (Phase 15)
		lockID := int64(crc32.ChecksumIEEE([]byte(name)))
		if err := repo.AcquireLock(ctx, tx, lockID); err != nil {
			return err
		}

		// b. Idempotency Check
		var existingChecksum, status string
		err = tx.QueryRow(ctx, "SELECT checksum, status FROM cascata_migrations WHERE name = $1", name).Scan(&existingChecksum, &status)
		if err == nil {
			if status == domain.MigrationCompleted {
				if existingChecksum != checksum {
					return fmt.Errorf("checksum mismatch: migration %s already applied with different content", name)
				}
				slog.Debug("migration: skipping already applied change", "name", name)
				return nil
			}
		}

		// c. Pre-Execution Record (Audit Fidelity)
		m := &domain.Migration{
			Slug:     slug,
			Name:     name,
			DDL:      ddl,
			Checksum: checksum,
			Status:   domain.MigrationRunning,
		}
		repo.RecordStep(ctx, tx, m)

		// d. Handle Concurrent Operations (Non-Transactional DDL)
		if isConcurrent {
			return fmt.Errorf("migration: CONCURRENTLY operations not supported in automated batch (Manual migration required)")
		}

		// e. Execute DDL
		start := time.Now()
		if _, err := tx.Exec(ctx, ddl); err != nil {
			m.Status = domain.MigrationFailed
			m.Error = err.Error()
			m.AppliedAt = time.Now()
			repo.RecordStep(ctx, tx, m)
			return fmt.Errorf("ddl execution failed: %w", err)
		}

		// f. Mark as COMPLETED
		m.Status = domain.MigrationCompleted
		m.AppliedAt = time.Now()
		repo.RecordStep(ctx, tx, m)

		// g. Audit Integration (Sinergy Phase 19/22)
		entry := &domain.AuditEntry{
			Project:   slug,
			Operation: "MIGRATION_APPLIED",
			Table:     "cascata_migrations",
			ActorType: "SYSTEM",
			Payload:   fmt.Sprintf("Migration %s executed in %v", name, time.Since(start)),
		}
		s.audit.WriteEntry(ctx, entry)
		
		return nil
	})

	if err != nil {
		span.RecordError(err)
		slog.Error("migration: final failure", "name", name, "err", err)
		return err
	}

	slog.Info("migration: successfully completed", "name", name, "slug", slug)
	return nil
}

func (s *MigrationService) calculateChecksum(ddl string) string {
	h := sha256.New()
	h.Write([]byte(ddl))
	return hex.EncodeToString(h.Sum(nil))
}
