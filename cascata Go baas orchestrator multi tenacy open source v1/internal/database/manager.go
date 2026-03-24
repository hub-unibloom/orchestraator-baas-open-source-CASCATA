package database

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"cascata/internal/domain"
	"cascata/internal/telemetry"
)

// TenantPoolManager is now a lightweight Logical Multiplexer.
// It abolishes the need to map physical postgres pools per tenant, avoiding 50GB RAM usages.
// Instead, it routes all queries through the master pool, trusting WithRLS to enforce isolation.
type TenantPoolManager struct {
	masterRepo    *Repository
	telemetry     *telemetry.TelemetryEngine
	auditSvc      domain.Auditor
	
	// BYOD Strategy: We cache external pools for premium tenants.
	externalPools map[string]*Repository
	mu            sync.RWMutex
}

// NewTenantPoolManager initializes the zero-overhead logical pool multiplexer.
func NewTenantPoolManager(masterRepo *Repository, tele *telemetry.TelemetryEngine, audit domain.Auditor) *TenantPoolManager {
	return &TenantPoolManager{
		masterRepo:    masterRepo,
		telemetry:     tele,
		auditSvc:      audit,
		externalPools: make(map[string]*Repository),
	}
}

// GetPool identifies if the project lives in the shared cluster or an external BYOD source.
func (m *TenantPoolManager) GetPool(ctx context.Context, slug, dbName string, tenantMax int) (*Repository, error) {
	// 1. Check Metadata for External DB bypass (BYOD)
	// In production, we resolution service (ProjectService) would pass the external DNS here.
	// For now, we routing logic: If dbName starts with 'external:', we treat as BYOD.
	
	const byodPrefix = "external:"
	if !strings.HasPrefix(dbName, byodPrefix) {
		// Standard Tenant: Return the high-density Master Pool
		return m.masterRepo, nil
	}

	externalURL := strings.TrimPrefix(dbName, byodPrefix)
	
	m.mu.RLock()
	p, ok := m.externalPools[slug]
	m.mu.RUnlock()
	
	if ok { return p, nil }

	// 2. Instantiate and Cache External Pool (Lazy Loading)
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Double check
	if p, ok := m.externalPools[slug]; ok { return p, nil }

	slog.Info("pool_manager: initializing dedicated BYOD pool", "slug", slug)
	
	repo, err := Connect(ctx, externalURL)
	if err != nil {
		return nil, fmt.Errorf("pool_manager: failed to bridge external db: %w", err)
	}
	
	m.externalPools[slug] = repo
	return repo, nil
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
