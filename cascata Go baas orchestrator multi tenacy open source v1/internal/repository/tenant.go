package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"

	"cascata/internal/database"
)

// TenantRepository manages the creation and physical lifecycle of tenant databases.
type TenantRepository struct {
	repo *database.Repository
}

// NewTenantRepository initializes the physical pool manager.
func NewTenantRepository(repo *database.Repository) *TenantRepository {
	return &TenantRepository{repo: repo}
}

// ProvisionNewDatabase creates a physical Postgres database for a tenant.
func (r *TenantRepository) ProvisionNewDatabase(ctx context.Context, dbName string) error {
	// 1. Sanitize dbName to prevent injection in CREATE DATABASE command.
	// Primary rule: alphanumeric and underscores only.
	re := regexp.MustCompile(`^[a-z0-9_]+$`)
	if !re.MatchString(dbName) {
		return errors.New("repository.Tenant.Provision: invalid database name pattern")
	}

	// 2. Execute CREATE DATABASE.
	// NOTE: pgxpool.Exec can run this as long as it's outside of pgx.BeginFunc.
	sql := fmt.Sprintf("CREATE DATABASE %s", dbName)

	_, err := r.repo.Pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("repository.Tenant.Provision: create db: %w", err)
	}

	slog.Info("physical database provisioned", "dbName", dbName)
	return nil
}

// DropDatabase removes a physical tenant database. (Security-Gated)
func (r *TenantRepository) DropDatabase(ctx context.Context, dbName string) error {
	re := regexp.MustCompile(`^[a-z0-9_]+$`)
	if !re.MatchString(dbName) {
		return errors.New("repository.Tenant.Drop: invalid dbName")
	}

	sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
	_, err := r.repo.Pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("repository.Tenant.Drop: failure: %w", err)
	}

	slog.Warn("physical database DROPPED", "dbName", dbName)
	return nil
}
