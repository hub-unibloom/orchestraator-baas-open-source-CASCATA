package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/repository"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SyncService orchestrates the convergent data synchronization (CRDT).
// It ensures that distributed clients (web, mobile, offline-first) reach a consistent state using LWW-Element-Set.
type SyncService struct {
	poolMgr *database.TenantPoolManager
	repo    *repository.SyncRepository
	audit   *AuditService
}

func NewSyncService(poolMgr *database.TenantPoolManager, audit *AuditService) *SyncService {
	return &SyncService{
		poolMgr: poolMgr,
		repo:    repository.NewSyncRepository(),
		audit:   audit,
	}
}

// MergeBatch processes a set of local changes from a client, 
// resolving conflicts based on the Last-Write-Wins (LWW) strategy.
// Integrated with OpenTelemetry for batch tracing.
func (s *SyncService) MergeBatch(ctx context.Context, payload *domain.SyncPayload) ([]domain.SyncResult, error) {
	tr := otel.Tracer("sync-engine")
	ctx, span := tr.Start(ctx, fmt.Sprintf("sync: %s", payload.Table), trace.WithAttributes(
		attribute.String("cascata.project", payload.ProjectSlug),
		attribute.String("sync.table", payload.Table),
		attribute.Int("sync.count", len(payload.Operations)),
	))
	defer span.End()

	slog.Info("sync: starting merge operation", "table", payload.Table, "slug", payload.ProjectSlug)
	
	pool, err := s.poolMgr.GetPool(ctx, payload.ProjectSlug)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("sync.MergeBatch: pool unreachable: %w", err)
	}

	results := make([]domain.SyncResult, 0, len(payload.Operations))

	// Execute Batch in unique transaction for atomicity (Phase 11.1)
	err = pgx.BeginFunc(ctx, pool.Pool, func(tx pgx.Tx) error {
		for _, op := range payload.Operations {
			res, err := s.processRecord(ctx, tx, payload.Table, op)
			if err != nil {
				slog.Warn("sync: record process failure", "id", op.ID, "error", err)
				results = append(results, domain.SyncResult{ID: op.ID, Status: "error", ServerTime: time.Now()})
				continue
			}
			results = append(results, res)
		}
		
		// Audit the entry (Sinergy Phase 19)
		entry := &domain.AuditEntry{
			Project:   payload.ProjectSlug,
			Operation: "SYNC_BATCH_MERGED",
			Table:     payload.Table,
			ActorType: "SYSTEM_SYNC",
			Payload:   fmt.Sprintf("Processed %d operations for table %s", len(payload.Operations), payload.Table),
		}
		s.audit.WriteEntry(ctx, entry)

		return nil
	})

	return results, err
}

// processRecord applies a single change if it wins the LWW timeline.
func (s *SyncService) processRecord(ctx context.Context, tx pgx.Tx, table string, op domain.SyncOperation) (domain.SyncResult, error) {
	// 1. Fetch Server-Side Metadata with 'FOR UPDATE' (Repository Sinergy)
	serverUpdatedAt, exists, err := s.repo.GetRecordMetadata(ctx, tx, table, op.ID)
	if err != nil {
		return domain.SyncResult{}, err
	}

	// 2. LWW Resolution
	// If record exists and incoming change is older, we REJECT it.
	if exists {
		if op.UpdatedAt.Before(serverUpdatedAt) || op.UpdatedAt.Equal(serverUpdatedAt) {
			slog.Debug("sync: change rejected (older version)", "id", op.ID, "client", op.UpdatedAt, "server", serverUpdatedAt)
			return domain.SyncResult{ID: op.ID, Status: "rejected_older", ServerTime: serverUpdatedAt}, nil
		}
	}

	// 3. Apply the merger (Idempotent Upsert)
	if err := s.repo.UpsertRecord(ctx, tx, table, op); err != nil {
		return domain.SyncResult{}, err
	}

	return domain.SyncResult{ID: op.ID, Status: "merged", ServerTime: time.Now()}, nil
}
