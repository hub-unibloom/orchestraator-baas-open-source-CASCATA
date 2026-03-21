package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"cascata/internal/config"
	"cascata/internal/database"
)

// MigrationService handles the execution of schema templates on tenant databases.
type MigrationService struct {
	Cfg *config.Config
}

// NewMigrationService initializes the schema manager.
func NewMigrationService(cfg *config.Config) *MigrationService {
	return &MigrationService{Cfg: cfg}
}

// BootstrapTenant applies the standard apartment template to a new database.
func (s *MigrationService) BootstrapTenant(ctx context.Context, dbName string) error {
	slog.Info("migration: bootstrapping tenant", "dbName", dbName)

	// 1. Establish tenant connection pool.
	tenantRepo, err := database.NewTenantPool(ctx, s.Cfg.DatabaseURL, dbName)
	if err != nil {
		return fmt.Errorf("service.Migration.Bootstrap: connect: %w", err)
	}
	defer tenantRepo.Close()

	// 2. Read the standard template.
	// Path should be relative to project root or absolute if configured.
	templatePath := "internal/templates/tenant_schema.sql"
	sql, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("service.Migration.Bootstrap: read template: %w", err)
	}

	// 3. Execute the template as a single block.
	_, err = tenantRepo.Pool.Exec(ctx, string(sql))
	if err != nil {
		return fmt.Errorf("service.Migration.Bootstrap: execute: %w", err)
	}

	slog.Info("tenant bootstrap successful", "dbName", dbName)
	return nil
}
