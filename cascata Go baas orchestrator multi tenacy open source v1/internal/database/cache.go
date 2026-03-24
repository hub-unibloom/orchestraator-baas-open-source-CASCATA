package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheManager handles the read-through and write-around caching logic via Dragonfly.
// It acts as the "Hyper-Drive" of the Cascata Read Path (Phase 12).
type CacheManager struct {
	dfly *redis.Client
}

func NewCacheManager(dfly *redis.Client) *CacheManager {
	return &CacheManager{dfly: dfly}
}

// BuildKey generates a deterministic cache key for a specific query within a tenant.
func (c *CacheManager) BuildKey(projectSlug, table, queryHash string) string {
	return fmt.Sprintf("cascata:cache:%s:%s:%s", projectSlug, table, queryHash)
}

// HashQuery creates a unique identifier for a SQL query and its parameters.
func (c *CacheManager) HashQuery(sql string, params []interface{}) string {
	h := sha256.New()
	h.Write([]byte(sql))
	h.Write([]byte(fmt.Sprintf("%v", params)))
	return hex.EncodeToString(h.Sum(nil))
}

// Get attempts to retrieve a JSON result from the distributed cache.
func (c *CacheManager) Get(ctx context.Context, key string) ([]byte, error) {
	return c.dfly.Get(ctx, key).Bytes()
}

// Set stores a JSON result in the cache with a default 10-minute TTL.
func (c *CacheManager) Set(ctx context.Context, key string, data []byte) error {
	return c.dfly.Set(ctx, key, data, 10*time.Minute).Err()
}

// InvalidateTable clears all cached queries for a specific table in a project.
// Uses Dragonfly Sets to track keys for O(1) mass invalidation (Phase 20 Acceleration).
func (c *CacheManager) InvalidateTable(ctx context.Context, projectSlug, table string) error {
	setKey := fmt.Sprintf("cascata:cache_index:%s:%s", projectSlug, table)
	
	// 1. Get all tracked keys for this table
	keys, err := c.dfly.SMembers(ctx, setKey).Result()
	if err != nil { return err }

	if len(keys) > 0 {
		// 2. Atomic delete of cached entries
		_ = c.dfly.Del(ctx, keys...).Err()
	}

	// 3. Purge the tracking set
	return c.dfly.Del(ctx, setKey).Err()
}

// TrackKey adds a key to the list of entries for a specific project/table.
func (c *CacheManager) TrackKey(ctx context.Context, projectSlug, table, key string) {
	setKey := fmt.Sprintf("cascata:cache_index:%s:%s", projectSlug, table)
	_ = c.dfly.SAdd(ctx, setKey, key).Err()
	_ = c.dfly.Expire(ctx, setKey, 1*time.Hour)
}
