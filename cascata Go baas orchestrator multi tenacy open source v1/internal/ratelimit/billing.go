package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// BillingQuotaManager orchestrates the "Economic Sinergy" of Cascata (Phase 22).
// It connects technical limits (Requests/Month) to business reality (Stripe/Billing).
type BillingQuotaManager struct {
	dfly *redis.Client
}

func NewBillingQuotaManager(dfly *redis.Client) *BillingQuotaManager {
	return &BillingQuotaManager{dfly: dfly}
}

// CheckLimit verifies if a tenant has exceeded their monthly quota based on their subscription tier.
func (m *BillingQuotaManager) CheckLimit(ctx context.Context, projectSlug string, limit int64) (bool, error) {
	key := fmt.Sprintf("cascata:billing:quota:%s:%s", projectSlug, time.Now().Format("2006-01"))

	// Atomic Increment and check
	count, err := m.dfly.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("billing: quota check failed: %w", err)
	}

	// Set expiration at end of month if new key
	if count == 1 {
		m.dfly.Expire(ctx, key, 32*24*time.Hour)
	}

	if count > limit {
		slog.Warn("billing: quota exceeded for project", "slug", projectSlug, "count", count, "limit", limit)
		return false, nil
	}

	return true, nil
}

// ReportUsage provides the telemetry for Stripe Metered Billing (Phase 22).
func (m *BillingQuotaManager) ReportUsage(ctx context.Context, projectSlug string, increment int64) error {
	// In production, this would queue a message for a Stripe usage report worker.
	slog.Debug("billing: reporting metered usage", "slug", projectSlug, "increment", increment)
	return nil
}
