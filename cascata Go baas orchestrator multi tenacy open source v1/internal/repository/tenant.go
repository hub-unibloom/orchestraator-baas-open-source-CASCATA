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

// ProvisionNewSchema creates a logical Postgres schema for a tenant as their "public" home.
func (r *TenantRepository) ProvisionNewSchema(ctx context.Context, slug string) error {
	// 1. Sanitize slug to prevent injection in CREATE SCHEMA command.
	re := regexp.MustCompile(`^[a-z0-9_]+$`)
	if !re.MatchString(slug) {
		return errors.New("repository.Tenant.Provision: invalid schema name pattern")
	}

	// 2. Execute CREATE SCHEMA with concatenative namespace (Sovereign Barrier).
	// We start with {slug}_public as the default home for the tenant.
	schemaHome := fmt.Sprintf("%s_public", slug)
	sql := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS \"%s\"", schemaHome)

	_, err := r.repo.Pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("repository.Tenant.Provision: create schema: %w", err)
	}

	slog.Info("sovereign schema home provisioned for tenant", "schema", schemaHome)
	return nil
}

// DropSchema removes a logical tenant schema and all its objects. (Security-Gated)
func (r *TenantRepository) DropSchema(ctx context.Context, slug string) error {
	re := regexp.MustCompile(`^[a-z0-9_]+$`)
	if !re.MatchString(slug) {
		return errors.New("repository.Tenant.Drop: invalid schema name")
	}

	// CASCADE ensures all tables, functions, and RLS policies within the schema die with it.
	sql := fmt.Sprintf("DROP SCHEMA IF EXISTS \"%s\" CASCADE", slug)
	_, err := r.repo.Pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("repository.Tenant.Drop: failure: %w", err)
	}

	slog.Warn("logical schema DROPPED", "schema", slug)
	return nil
}
