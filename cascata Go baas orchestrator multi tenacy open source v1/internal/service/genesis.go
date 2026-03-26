package service

import (
	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/repository"
	"cascata/internal/security"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	pgx "github.com/jackc/pgx/v5"
)

// GenesisService orchestrates the physical creation and initialization of new Tenants.
// It is the midwife of the Cascata ecosystem, handling from DB creation to RLS hardening.
type GenesisService struct {
	repo         *database.Repository // Master pool (Administrative access)
	projectRepo  *repository.ProjectRepository
	tenantRepo   *repository.TenantRepository
	poolMgr      *database.TenantPoolManager
	security     *security.SecurityService
	migrationSvc *MigrationService
}

func NewGenesisService(repo *database.Repository, projectRepo *repository.ProjectRepository, tenantRepo *repository.TenantRepository, poolMgr *database.TenantPoolManager, security *security.SecurityService, migrationSvc *MigrationService) *GenesisService {
	return &GenesisService{
		repo:         repo,
		projectRepo:  projectRepo,
		tenantRepo:   tenantRepo,
		poolMgr:      poolMgr,
		security:     security,
		migrationSvc: migrationSvc,
	}
}

// CreateProject performs the full multi-step "Birth" process with Rollback safety (Phase 2-4 Synergy).
func (s *GenesisService) CreateProject(ctx context.Context, p *domain.Project) error {
	slog.Info("genesis: starting project creation", "slug", p.Slug, "db", p.DBName)

	// 1. Logical Schema Creation (Sanitized via Repository)
	if err := s.tenantRepo.ProvisionNewSchema(ctx, p.Slug); err != nil {
		return fmt.Errorf("genesis: create schema failed: %w", err)
	}

	// Atomic Cleanup: If any step fails, we must purge all provisioned artifacts
	success := false
	defer func() {
		if !success {
			slog.Warn("genesis: rolling back project creation due to failure", "slug", p.Slug)
			// 1. Purge DB Schema
			_ = s.tenantRepo.DropSchema(context.Background(), p.Slug)
			// 2. Clear Metadata in DB (Only if it was created)
			_ = s.repo.WithRLS(context.Background(), database.UserClaims{Role: "service_role"}, "cascata", false, func(tx pgx.Tx) error {
				return s.projectRepo.Delete(context.Background(), tx, p.Slug)
			})
		}
	}()

	// 2. Initialize and Encrypt Secrets in Native Security Engine (AES-256-GCM)
	if p.JWTSecret == "" {
		p.JWTSecret = s.generateRandomKey(32)
	}
	if p.AnonKey == "" {
		p.AnonKey = "ak_" + s.generateRandomKey(24)
	}
	if p.ServiceKey == "" {
		p.ServiceKey = "sk_" + s.generateRandomKey(32)
	}

	// Encrypt Core Credentials BEFORE registration
	if encryptedJWT, err := s.security.Encrypt(ctx, p.JWTSecret); err == nil {
		p.JWTSecret = encryptedJWT
	}
	if encryptedAnon, err := s.security.Encrypt(ctx, p.AnonKey); err == nil {
		p.AnonKey = encryptedAnon
	}
	if encryptedService, err := s.security.Encrypt(ctx, p.ServiceKey); err == nil {
		p.ServiceKey = encryptedService
	}

	// Encrypt Secondary Secret (Agency Mode)
	if p.SecondarySecretHash != "" {
		if encryptedSec, err := s.security.Encrypt(ctx, p.SecondarySecretHash); err == nil {
			p.SecondarySecretHash = encryptedSec
		}
	}

	// 3. System Metadata Registration (Must exist for Migration Engine to resolve)
	// Sovereign Registration: Metadata must be stored in cascata_system.
	err := s.repo.WithRLS(ctx, database.UserClaims{Role: "service_role"}, "cascata", false, func(tx pgx.Tx) error {
		if err := s.projectRepo.Create(ctx, tx, p); err != nil {
			return fmt.Errorf("genesis: metadata registration failed: %w", err)
		}
		return nil
	})

	if err != nil {
		slog.Error("genesis: failed to register project metadata", "slug", p.Slug, "err", err)
		return err
	}

	// 4. Apply Initial Schema & Hardening (Phase 15 Sinergy)
	if err := s.initializeSchema(ctx, p.Slug); err != nil {
		return fmt.Errorf("genesis: schema initialization failed: %w", err)
	}

	success = true
	slog.Info("genesis: project created successfully", "slug", p.Slug)
	return nil
}

// DeleteProject performs the final "Death" process:
func (s *GenesisService) DeleteProject(ctx context.Context, slug string) error {
	var p *domain.Project
	var err error

	// 1. Resolve project within Sovereign Context
	err = s.repo.WithRLS(ctx, database.UserClaims{Role: "service_role"}, "cascata", false, func(tx pgx.Tx) error {
		p, err = s.projectRepo.GetByIdentifier(ctx, tx, slug)
		return err
	})
	if err != nil {
		return fmt.Errorf("genesis: cannot resolve project for deletion: %w", err)
	}

	slog.Info("genesis: starting project deletion", "slug", slug)

	// 2. Termination: Drop Logical Schema (Antifragile)
	if err := s.tenantRepo.DropSchema(ctx, p.Slug); err != nil {
		slog.Warn("genesis: drop schema failed during deletion (might already be gone)", "schema", p.Slug, "err", err)
	}

	// 3. System Metadata Purge within Sovereign Context
	err = s.repo.WithRLS(ctx, database.UserClaims{Role: "service_role"}, "cascata", false, func(tx pgx.Tx) error {
		return s.projectRepo.Delete(ctx, tx, slug)
	})
	
	slog.Info("genesis: project deleted successfully", "slug", slug)
	return err
}

func (s *GenesisService) initializeSchema(ctx context.Context, slug string) error {
	const baseDDL = `
		-- Ensure clean auth namespace
		-- Cascata System Functions for RLS context
		CREATE OR REPLACE FUNCTION current_project_slug() RETURNS text AS $$
			SELECT current_setting('cascata.project_slug', true);
		$$ LANGUAGE sql STABLE;

		-- Auth Helpers (Phase 10.1: Context Aware)
		CREATE OR REPLACE FUNCTION uid() RETURNS text AS $$
			SELECT current_setting('request.jwt.claim.sub', true);
		$$ LANGUAGE sql STABLE;

		CREATE OR REPLACE FUNCTION role() RETURNS text AS $$
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

		-- Resident Auth Table (Phase 11 Sync Foundation + Phase 10.1 Lexicon)
		CREATE TABLE IF NOT EXISTS users (
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

		-- Apply Real-time Trigger to Users
		CREATE TRIGGER users_realtime_notify
		AFTER INSERT OR UPDATE OR DELETE ON users
		FOR EACH ROW EXECUTE FUNCTION cascata_notify_realtime();

		ALTER TABLE users ENABLE ROW LEVEL SECURITY;
	`
	
	// Delegate to MigrationService for atomic and auditable schema application
	return s.migrationSvc.ApplyMigration(ctx, slug, "0001_genesis_foundation", baseDDL)
}

// generateRandomKey produces high-entropy strings for zero-trust credentials (JWT Secrets, Anon Keys).
func (s *GenesisService) generateRandomKey(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "insecure_fallback_key" // Should never happen with proper CSPRNG
	}
	return hex.EncodeToString(b)
}
