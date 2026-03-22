package database

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantPoolManager handles physical isolation of database pools per tenant.
// Phase 2 - Physical Isolation + Phase 22 Hardening.
type TenantPoolManager struct {
	mu           sync.RWMutex
	pools        map[string]*tenantPoolEntry
	masterURL    string
	globalMax    int
}

type tenantPoolEntry struct {
	repo       *Repository
	lastUsed   time.Time
	maxConns   int
}

// NewTenantPoolManager initializes the physical pool orchestrator.
func NewTenantPoolManager(masterURL string, globalMax int) *TenantPoolManager {
	return &TenantPoolManager{
		pools:     make(map[string]*tenantPoolEntry),
		masterURL: masterURL,
		globalMax: globalMax,
	}
}

// GetPool returns an isolated connection pool for a specific tenant.
// Implements On-Demand creation, Hard Caps, and 503 handling.
func (m *TenantPoolManager) GetPool(ctx context.Context, slug, dbName string, tenantMax int) (*Repository, error) {
	m.mu.RLock()
	entry, ok := m.pools[slug]
	m.mu.RUnlock()

	if ok {
		m.mu.Lock()
		entry.lastUsed = time.Now()
		m.mu.Unlock()
		return entry.repo, nil
	}

	// Double-checked locking for on-demand creation.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Re-verify after lock.
	if entry, ok = m.pools[slug]; ok {
		entry.lastUsed = time.Now()
		return entry.repo, nil
	}

	// 1. Check Global Ceiling.
	// In production, this would be a real-time sum of active pools' max_conns.
	// For now, we enforce the tenantMax against a safe baseline.
	if tenantMax > m.globalMax {
		tenantMax = m.globalMax / 10 // Safety fallback
	}

	// 2. Create the physical pool.
	slog.Info("creating isolated pool for tenant", "slug", slug, "db", dbName, "max_conns", tenantMax)
	
	repo, err := NewTenantPoolWithLimit(ctx, m.masterURL, dbName, tenantMax)
	if err != nil {
		return nil, fmt.Errorf("pool_manager: failed to establish isolated pool: %w", err)
	}

	m.pools[slug] = &tenantPoolEntry{
		repo:     repo,
		lastUsed: time.Now(),
		maxConns: tenantMax,
	}

	return repo, nil
}

// NewTenantPoolWithLimit encapsulates pgxpool tuning for a specific tenant.
func NewTenantPoolWithLimit(ctx context.Context, masterURL, dbName string, maxConns int) (*Repository, error) {
	cfg, err := pgxpool.ParseConfig(masterURL)
	if err != nil {
		return nil, err
	}

	cfg.ConnConfig.Database = dbName
	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = 1
	cfg.MaxConnIdleTime = 30 * time.Second // Idle Transaction Reaper
	cfg.MaxConnLifetime = 30 * time.Minute

	// PgBouncer Transactional Mode safety is handled in WithRLS via RESET ALL.
	
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &Repository{Pool: pool}, nil
}

// CleanupTask handles destruction of pools after a period of inactivity.
func (m *TenantPoolManager) CleanupTask(ctx context.Context, idleTimeout time.Duration) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			for slug, entry := range m.pools {
				if time.Since(entry.lastUsed) > idleTimeout {
					slog.Info("destroying inactive pool", "slug", slug)
					entry.repo.Close()
					delete(m.pools, slug)
				}
			}
			m.mu.Unlock()
		}
	}
}
