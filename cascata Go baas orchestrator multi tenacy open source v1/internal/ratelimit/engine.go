package ratelimit

import (
	"context"
	"fmt"
	"time"

	"cascata/internal/database"
	"github.com/redis/go-redis/v9"
)

// AdaptiveEngine manages hierarchical quotas and CPU-aware limits.
type AdaptiveEngine struct {
	redis      *redis.Client
	systemPool *database.Pool
}

func NewAdaptiveEngine(rdb *redis.Client, systemPool *database.Pool) *AdaptiveEngine {
	return &AdaptiveEngine{redis: rdb, systemPool: systemPool}
}

// Config maps the limits for a specific tier/group.
type Config struct {
	Limit          int
	WindowSeconds  int
	SpeedPctOnNerf int
}

// Check verifies if the requested operation can proceed under current quotas and load.
func (e *AdaptiveEngine) Check(ctx context.Context, slug, ip, method string, cfg Config, isNerfed bool) (bool, error) {
	// 1. CPU Backpressure: Check Global Health (Mocked L1/L2 read)
	mult := e.getCPUMultiplier(ctx)

	// 2. Nerf System: Grace period reduction
	limit := float64(cfg.Limit) * mult
	if isNerfed {
		limit = limit * (float64(cfg.SpeedPctOnNerf) / 100.0)
		if limit < 1 {
			limit = 1
		}
	}

	// 3. Operation Weights
	// Reads = 1, Writes = 5, Deletes = 3
	weight := 1
	switch method {
	case "POST", "PUT", "PATCH":
		weight = 5
	case "DELETE":
		weight = 3
	}

	// 4. Redis Script execution (Token Bucket or Fixed Window)
	// Here we implement an atomic Lua-based token deduction supporting Weights
	key := fmt.Sprintf("rl:%s:%s", slug, ip)
	
	allowed, err := e.deductTokens(ctx, key, int(limit), cfg.WindowSeconds, weight)
	if err != nil {
		return false, err
	}

	return allowed, nil
}

// deductTokens uses Redis INCR + EXPIRE tracking to subtract multi-weight tokens
func (e *AdaptiveEngine) deductTokens(ctx context.Context, key string, limit, window, weight int) (bool, error) {
	pipe := e.redis.TxPipeline()
	ctr := pipe.IncrBy(ctx, key, int64(weight))
	pipe.Expire(ctx, key, time.Duration(window)*time.Second)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	return ctr.Val() <= int64(limit), nil
}

func (e *AdaptiveEngine) getCPUMultiplier(ctx context.Context) float64 {
	// Reads latest CPU metric pushed to Dragonfly by telemetry agent.
	// E.g., if load > 90%, returns 0.5 (halving all limits to prevent crash)
	val, err := e.redis.Get(ctx, "sys:health:cpu_load").Float64()
	if err != nil { return 1.0 }

	if val > 90.0 {
		return 0.5
	} else if val > 75.0 {
		return 0.8
	}
	return 1.0
}

// ValidateHierarchicalQuota forces the Master decision if tenant tries to consume more than global limits
func (e *AdaptiveEngine) ValidateHierarchicalQuota(ctx context.Context, tenantSlug string, requestedConnections int) error {
	// Master defined global ceiling
	const globalMaxConnections = 10000

	// Get sum of all tenants
	var totalAllocated int
	err := e.systemPool.QueryRow(ctx, "SELECT COALESCE(SUM(max_connections), 0) FROM system.projects").Scan(&totalAllocated)
	if err != nil { return err }

	if totalAllocated + requestedConnections > globalMaxConnections {
		return fmt.Errorf("hierarchical quota violation: global limit of %d reached. Master reduction required.", globalMaxConnections)
	}
	return nil
}
