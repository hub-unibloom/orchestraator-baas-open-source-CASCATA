package database

import (
	"context"
	"log/slog"
	"time"

	"cascata/internal/domain"
	"cascata/internal/telemetry"
)

// TenantPoolManager is now a lightweight Logical Multiplexer.
// It abolishes the need to map physical postgres pools per tenant, avoiding 50GB RAM usages.
// Instead, it routes all queries through the master pool, trusting WithRLS to enforce isolation.
type TenantPoolManager struct {
	masterRepo *Repository
	telemetry  *telemetry.TelemetryEngine
	auditSvc   domain.Auditor
}

// NewTenantPoolManager initializes the zero-overhead logical pool multiplexer.
func NewTenantPoolManager(masterRepo *Repository, tele *telemetry.TelemetryEngine, audit domain.Auditor) *TenantPoolManager {
	return &TenantPoolManager{
		masterRepo: masterRepo,
		telemetry:  tele,
		auditSvc:   audit,
	}
}

// GetPool simply returns the highly-concurrent master pool.
// The true physical isolation is enforced precisely inside the WithRLS transaction boundaries.
func (m *TenantPoolManager) GetPool(ctx context.Context, slug, dbName string, tenantMax int) (*Repository, error) {
	return m.masterRepo, nil
}

// CheckPoolHealth checks the health of the entire logical ecosystem.
func (m *TenantPoolManager) CheckPoolHealth(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Just pinging the master pool to know the heart is beating
			if err := m.masterRepo.Pool.Ping(ctx); err != nil {
				slog.Error("pool_manager: critical master pool failure", "error", err)
			}
		}
	}
}

// CleanupTask is kept for interface compatibility but does nothing,
// as the central pgxpool handles its own idle connection reaping.
func (m *TenantPoolManager) CleanupTask(ctx context.Context, idleTimeout time.Duration) {
	// A no-op. The central pool manages itself automatically.
}
