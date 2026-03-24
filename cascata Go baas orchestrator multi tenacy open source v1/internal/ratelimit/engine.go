package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"cascata/internal/database"
	"github.com/redis/go-redis/v9"
)

// AdaptiveEngine manages hierarchical quotas and CPU-aware limits via DragonflyDB.
// Support for Phase 3 - Security & Resilience.
type AdaptiveEngine struct {
	dfly       *redis.Client
	systemPool *database.Repository
	localBans  sync.Map // L1 Cache for ultra-fast security check (Phase 26)
}

// Config maps the limits for a specific tier/group.
type Config struct {
	Limit          int
	WindowSeconds  int
	SpeedPctOnNerf int
}

func NewAdaptiveEngine(dfly *redis.Client, systemPool *database.Repository) *AdaptiveEngine {
	e := &AdaptiveEngine{dfly: dfly, systemPool: systemPool}
	go e.listenForGlobalBans(context.Background())
	return e
}

func (e *AdaptiveEngine) listenForGlobalBans(ctx context.Context) {
	pubsub := e.dfly.Subscribe(ctx, "cascata:security:bans")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		slog.Debug("security: global ban received", "ip", msg.Payload)
		e.localBans.Store(msg.Payload, time.Now().Add(24*time.Hour))
	}
}

// Check verifies if the requested operation can proceed under current quotas and load.
func (e *AdaptiveEngine) Check(ctx context.Context, slug, ip, method string, cfg Config, isNerfed bool) (bool, error) {
	// 1. IP Strikes & Ban Check
	if banned, _ := e.IsIPBanned(ctx, ip); banned {
		return false, fmt.Errorf("SECURITY_ALARM: IP banned due to excessive violations")
	}

	// 2. CPU Backpressure: Check Global Health (Mocked L1/L2 read from Dragonfly)
	mult := e.getCPUMultiplier(ctx)

	// 3. Nerf System: Grace period reduction
	limit := float64(cfg.Limit) * mult
	if isNerfed {
		limit = limit * (float64(cfg.SpeedPctOnNerf) / 100.0)
		if limit < 1 {
			limit = 1
		}
	}

	// 4. Operation Weights
	// Reads = 1, Writes = 5, Deletes = 3
	weight := 1
	switch method {
	case "POST", "PUT", "PATCH":
		weight = 5
	case "DELETE":
		weight = 3
	}

	// 5. Atomic Token Deduction on Dragonfly
	key := fmt.Sprintf("rl:%s:%s", slug, ip)
	allowed, err := e.deductTokens(ctx, key, int(limit), cfg.WindowSeconds, weight)
	if err != nil {
		return false, err
	}

	if !allowed {
		// Log a strike for exhausting quota repeatedly
		_ = e.ReportIPStrike(ctx, ip)
	}

	return allowed, nil
}

// RecordLoginFailure tracks failed login attempts to trigger 30min locks.
func (e *AdaptiveEngine) RecordLoginFailure(ctx context.Context, identifier string) error {
	key := fmt.Sprintf("auth:fails:%s", identifier)
	pipe := e.dfly.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 15*time.Minute)
	res, err := pipe.Exec(ctx)
	if err != nil { return err }

	count := res[0].(*redis.IntCmd).Val()
	if count >= 5 {
		// Lock for 30 minutes
		lockKey := fmt.Sprintf("auth:locked:%s", identifier)
		e.dfly.Set(ctx, lockKey, "1", 30*time.Minute)
		slog.Warn("SECURITY: brute-force attack blocked", "identifier", identifier, "attempts", count)
	}
	return nil
}

// IsLoginLocked checks if a specific member/user is currently under lock.
func (e *AdaptiveEngine) IsLoginLocked(ctx context.Context, identifier string) (bool, time.Duration) {
	key := fmt.Sprintf("auth:locked:%s", identifier)
	ttl, _ := e.dfly.TTL(ctx, key).Result()
	return ttl > 0, ttl
}

// ReportIPStrike increments the strike counter for an IP.
func (e *AdaptiveEngine) ReportIPStrike(ctx context.Context, ip string) error {
	key := fmt.Sprintf("ip:strikes:%s", ip)
	pipe := e.dfly.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 1*time.Hour)
	res, err := pipe.Exec(ctx)
	if err != nil { return err }

	strikes := res[0].(*redis.IntCmd).Val()
	if strikes >= 10 {
		// Ban for 24 hours
		banKey := fmt.Sprintf("ip:banned:%s", ip)
		e.dfly.Set(ctx, banKey, "1", 24*time.Hour)
		
		// Propagate to all workers (Phase 26)
		e.dfly.Publish(ctx, "cascata:security:bans", ip)
		
		slog.Error("SECURITY_ALARM: IP auto-banned and propagated", "ip", ip, "strikes", strikes)
	}
	return nil
}

// IsIPBanned verifies if a request source is forbidden (Phase 26: L1 + L2 Check).
func (e *AdaptiveEngine) IsIPBanned(ctx context.Context, ip string) (bool, time.Duration) {
	// 1. Check L1 Cache (Local Map)
	if expiry, ok := e.localBans.Load(ip); ok {
		if time.Now().Before(expiry.(time.Time)) {
			return true, time.Until(expiry.(time.Time))
		}
		e.localBans.Delete(ip)
	}

	// 2. Fallback to L2 Cache (Dragonfly)
	key := fmt.Sprintf("ip:banned:%s", ip)
	ttl, _ := e.dfly.TTL(ctx, key).Result()
	
	if ttl > 0 {
		// Sync back to L1
		e.localBans.Store(ip, time.Now().Add(ttl))
		return true, ttl
	}

	return false, 0
}

// deductTokens uses Dragonfly INCR + EXPIRE tracking to subtract multi-weight tokens
func (e *AdaptiveEngine) deductTokens(ctx context.Context, key string, limit, window, weight int) (bool, error) {
	pipe := e.dfly.TxPipeline()
	_ = pipe.IncrBy(ctx, key, int64(weight))
	pipe.Expire(ctx, key, time.Duration(window)*time.Second)
	res, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	return res[0].(*redis.IntCmd).Val() <= int64(limit), nil
}

func (e *AdaptiveEngine) getCPUMultiplier(ctx context.Context) float64 {
	val, err := e.dfly.Get(ctx, "sys:health:cpu_load").Result()
	if err != nil { return 1.0 }

	f, _ := strconv.ParseFloat(val, 64)
	if f > 90.0 {
		return 0.5
	} else if f > 75.0 {
		return 0.8
	}
	return 1.0
}
