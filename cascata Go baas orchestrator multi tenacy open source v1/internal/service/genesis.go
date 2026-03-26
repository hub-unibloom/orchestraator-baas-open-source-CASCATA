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

// CreateProject performs the full multi-step "Birth" process with Sovereign Atomicity.
func (s *GenesisService) CreateProject(ctx context.Context, p *domain.Project) error {
	slog.Info("genesis: starting project creation", "slug", p.Slug, "db", p.DBName)

	// 1. Initialize and Encrypt Secrets in Native Security Engine (AES-256-GCM)
	if p.JWTSecret == "" { p.JWTSecret = s.generateRandomKey(32) }
	if p.AnonKey == "" { p.AnonKey = "ak_" + s.generateRandomKey(24) }
	if p.ServiceKey == "" { p.ServiceKey = "sk_" + s.generateRandomKey(32) }

	// Encrypt Core Credentials BEFORE registration
	if encryptedJWT, err := s.security.Encrypt(ctx, p.JWTSecret); err == nil { p.JWTSecret = encryptedJWT }
	if encryptedAnon, err := s.security.Encrypt(ctx, p.AnonKey); err == nil { p.AnonKey = encryptedAnon }
	if encryptedService, err := s.security.Encrypt(ctx, p.ServiceKey); err == nil { p.ServiceKey = encryptedService }

	if p.SecondarySecretHash != "" {
		if encryptedSec, err := s.security.Encrypt(ctx, p.SecondarySecretHash); err == nil {
			p.SecondarySecretHash = encryptedSec
		}
	}

	// 2. Atomic Physical Provisioning via Sovereign Namespace Barrier
	claims := database.UserClaims{Role: "service_role"}
	err := s.repo.WithRLS(ctx, claims, "cascata", false, func(tx pgx.Tx) error {
		// a. Provision System Metadata
		if err := s.projectRepo.Create(ctx, tx, p); err != nil {
			return fmt.Errorf("genesis: metadata creation failed: %w", err)
		}

		// b. Provision Physical Schema (Isolated by RLS/search_path strategy)
		if err := s.tenantRepo.ProvisionNewSchema(ctx, tx, p.Slug); err != nil {
			return fmt.Errorf("genesis: physical provisioning failed: %w", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	// 3. Post-Provisioning Lifecycle (Schema Migrations & Hardening)
	// Atomic Cleanup on failure
	success := false
	defer func() {
		if !success {
			slog.Warn("genesis: rolling back project lifecycle", "slug", p.Slug)
			_ = s.repo.WithRLS(context.Background(), claims, "cascata", false, func(tx pgx.Tx) error {
				_ = s.tenantRepo.DropSchema(context.Background(), tx, p.Slug)
				return s.projectRepo.Delete(context.Background(), tx, p.Slug)
			})
		}
	}()

	// Apply Initial Schema (Phase 15 Sinergy)
	if err := s.initializeSchema(ctx, p.Slug); err != nil {
		return fmt.Errorf("genesis: schema initialization failed: %w", err)
	}

	success = true
	slog.Info("genesis: project created successfully", "slug", p.Slug)
	return nil
}

// DeleteProject removes a project's metadata and physical schema with Sovereign Isolation.
func (s *GenesisService) DeleteProject(ctx context.Context, slug string) error {
	slog.Info("genesis: starting project deletion", "slug", slug)

	claims := database.UserClaims{Role: "service_role"}
	err := s.repo.WithRLS(ctx, claims, "cascata", false, func(tx pgx.Tx) error {
		// 1. Drop Logical Schema (CASCADE ensures cleanup)
		if err := s.tenantRepo.DropSchema(ctx, tx, slug); err != nil {
			slog.Warn("genesis: drop schema failed during deletion (continuing metadata purge)", "err", err)
		}

		// 2. Purge System Metadata
		return s.projectRepo.Delete(ctx, tx, slug)
	})

	if err == nil {
		slog.Info("genesis: project deleted successfully", "slug", slug)
	}
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
