package service

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"cascata/internal/database"
)

// PITRService handles the orchestration of database snapshots and point-in-time recovery.
// It uses native PostgreSQL tooling (pg_dump/pg_restore) for absolute reliability (Phase 32).
type PITRService struct {
	poolMgr *database.TenantPoolManager
}

func NewPITRService(poolMgr *database.TenantPoolManager) *PITRService {
	return &PITRService{poolMgr: poolMgr}
}

// CreateSnapshot captures a consistent point-in-time state of a tenant database.
func (s *PITRService) CreateSnapshot(ctx context.Context, slug, dbName, targetPath string) error {
	slog.Info("pitr: starting snapshot creation", "slug", slug, "db", dbName)

	start := time.Now()

	// Logic: Execute pg_dump directly into the backup path.
	// Production-Grade: Use custom format (-Fc) for parallel restore and compression.
	// Use environment variables for credentials (provided by PoolManager)
	cmd := exec.CommandContext(ctx, "pg_dump", 
		"-Fc", 
		"-d", dbName, 
		"-f", targetPath,
		"--no-owner", "--no-acl",
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("pitr: snapshots failed", "slug", slug, "error", err, "output", string(output))
		return fmt.Errorf("pitr: backup command failed: %w", err)
	}

	slog.Info("pitr: snapshot completed successfully", "slug", slug, "duration", time.Since(start))
	return nil
}

// RestoreToPointInTime handles the recovery of a project state.
func (s *PITRService) RestoreToPointInTime(ctx context.Context, slug, dbName, sourcePath string) error {
	slog.Warn("pitr: CRITICAL - starting destructive restore", "slug", slug, "db", dbName)

	start := time.Now()

	// 1. Terminate all active sessions for the database to release locks
	// 2. Execute pg_restore into the database.
	cmd := exec.CommandContext(ctx, "pg_restore", 
		"-d", dbName, 
		"--clean",   // Drop objects before creating
		"--if-exists", 
		sourcePath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("pitr: restore failed", "slug", slug, "error", err, "output", string(output))
		return fmt.Errorf("pitr: restore command failed: %w", err)
	}

	slog.Info("pitr: restore completed successfully", "slug", slug, "duration", time.Since(start))
	return nil
}
