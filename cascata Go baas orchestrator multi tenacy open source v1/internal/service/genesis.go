package service

import (
	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/repository"
	"cascata/internal/vault"
	"context"
	"fmt"
	"log/slog"
)

// GenesisService orchestrates the physical creation and initialization of new Tenants.
// It is the midwife of the Cascata ecosystem, handling from DB creation to RLS hardening.
type GenesisService struct {
	repo         *database.Repository // Master pool (Administrative access)
	projectRepo  *repository.ProjectRepository
	tenantRepo   *repository.TenantRepository
	poolMgr      *database.TenantPoolManager
	vault        *vault.VaultService
	migrationSvc *MigrationService
}

func NewGenesisService(repo *database.Repository, projectRepo *repository.ProjectRepository, tenantRepo *repository.TenantRepository, poolMgr *database.TenantPoolManager, vault *vault.VaultService, migrationSvc *MigrationService) *GenesisService {
	return &GenesisService{
		repo:         repo,
		projectRepo:  projectRepo,
		tenantRepo:   tenantRepo,
		poolMgr:      poolMgr,
		vault:        vault,
		migrationSvc: migrationSvc,
	}
}

// CreateProject performs the full multi-step "Birth" process with Rollback safety (Phase 2-4 Synergy).
func (s *GenesisService) CreateProject(ctx context.Context, p *domain.Project) error {
	slog.Info("genesis: starting project creation", "slug", p.Slug, "db", p.DBName)

	// 1. Physical Database Creation (Sanitized via Repository)
	if err := s.tenantRepo.ProvisionNewDatabase(ctx, p.DBName); err != nil {
		return fmt.Errorf("genesis: create db failed: %w", err)
	}

	// Atomic Cleanup: If any step below fails, we must drop the DB
	success := false
	defer func() {
		if !success {
			slog.Warn("genesis: rolling back project creation (dropping db)", "slug", p.Slug, "db", p.DBName)
			_ = s.tenantRepo.DropDatabase(context.Background(), p.DBName)
		}
	}()

	// 2. Initialize Secrets in HashiCorp Vault (Phase 4)
	if err := s.vault.ProvisionTenantVault(ctx, p.Slug); err != nil {
		return fmt.Errorf("genesis: vault provisioning failed: %w", err)
	}

	// 3. Apply Initial Schema & Hardening (Phase 15 Sinergy)
	if err := s.initializeSchema(ctx, p.Slug); err != nil {
		return fmt.Errorf("genesis: schema initialization failed: %w", err)
	}

	// 4. System Metadata Registration
	if err := s.projectRepo.Create(ctx, p); err != nil {
		slog.Error("genesis: failed to register project metadata", "slug", p.Slug, "err", err)
		return fmt.Errorf("genesis: metadata registration failed: %w", err)
	}

	success = true
	slog.Info("genesis: project created successfully", "slug", p.Slug)
	return nil
}

// DeleteProject performs the final "Death" process:
// 1. Drop Physical PostgreSQL Database.
// 2. Cleanup System Metadata.
// 3. Vault Cleanup (Tombstoning) - Phase 4.
func (s *GenesisService) DeleteProject(ctx context.Context, slug string) error {
	p, err := s.projectRepo.GetByIdentifier(ctx, slug)
	if err != nil {
		return fmt.Errorf("genesis: cannot resolve project for deletion: %w", err)
	}

	slog.Info("genesis: starting project deletion", "slug", slug, "db", p.DBName)

	// 1. Termination: Drop Database (Antifragile)
	if err := s.tenantRepo.DropDatabase(ctx, p.DBName); err != nil {
		slog.Warn("genesis: drop db failed during deletion (might already be gone)", "db", p.DBName, "err", err)
	}

	// 2. Vault Tombstoning (Historical record of destruction)
	_ = s.vault.WriteSecret(ctx, fmt.Sprintf("cascata/tenants/%s/config", slug), map[string]any{
		"deleted_at": time.Now().Format(time.RFC3339),
		"status":     "deleted",
	})

	// 3. System Metadata Purge
	err = s.projectRepo.Delete(ctx, slug)
	
	slog.Info("genesis: project deleted successfully", "slug", slug)
	return err
}

func (s *GenesisService) initializeSchema(ctx context.Context, slug string) error {
	const baseDDL = `
		-- Ensure clean auth namespace
		CREATE SCHEMA IF NOT EXISTS auth;
		
		-- Cascata System Functions for RLS context
		CREATE OR REPLACE FUNCTION current_project_slug() RETURNS text AS $$
			SELECT current_setting('cascata.project_slug', true);
		$$ LANGUAGE sql STABLE;

		CREATE OR REPLACE FUNCTION auth.uid() RETURNS text AS $$
			SELECT current_setting('request.jwt.claim.sub', true);
		$$ LANGUAGE sql STABLE;

		CREATE OR REPLACE FUNCTION auth.role() RETURNS text AS $$
			SELECT current_setting('request.jwt.claim.role', true);
		$$ LANGUAGE sql STABLE;

		-- Real-time Engine (Internal Trigger Function)
		CREATE OR REPLACE FUNCTION cascata_notify_realtime() RETURNS trigger AS $$
		DECLARE
		  payload JSONB;
		BEGIN
		  payload = jsonb_build_object(
		    'table', TG_TABLE_NAME,
		    'schema', TG_TABLE_SCHEMA,
		    'op', TG_OP,
		    'data', CASE 
		              WHEN TG_OP = 'DELETE' THEN to_jsonb(OLD) 
		              ELSE to_jsonb(NEW) 
		            END,
		    'old', CASE
		              WHEN TG_OP = 'UPDATE' THEN to_jsonb(OLD)
		              ELSE NULL
		            END,
		    'timestamp', now()
		  );
		  PERFORM pg_notify('cascata_realtime', payload::text);
		  RETURN COALESCE(NEW, OLD);
		END;
		$$ LANGUAGE plpgsql;

		-- Basic Extensions
		CREATE EXTENSION IF NOT EXISTS pgcrypto;
		CREATE EXTENSION IF NOT EXISTS pg_graphql;

		-- Resident Auth Table (Phase 11 Sync Foundation + Phase 10.1 Lexicon)
		CREATE TABLE IF NOT EXISTS auth.users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT UNIQUE,
			cpf TEXT UNIQUE,
			whatsapp TEXT UNIQUE,
			password_hash TEXT,
			role TEXT DEFAULT 'authenticated',
			is_email_verified BOOLEAN DEFAULT false,
			is_phone_verified BOOLEAN DEFAULT false,
			metadata JSONB DEFAULT '{}',
			last_login TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		);
		ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
	`
	
	// Delegate to MigrationService for atomic and auditable schema application
	return s.migrationSvc.ApplyMigration(ctx, slug, "0001_genesis_foundation", baseDDL)
}
