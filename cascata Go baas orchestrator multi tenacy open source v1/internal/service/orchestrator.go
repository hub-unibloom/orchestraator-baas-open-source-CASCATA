package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"cascata/internal/domain"
	"cascata/internal/repository"
	"cascata/internal/vault"
)

// OrchestratorService coordinates tenant creation lifecycles.
type OrchestratorService struct {
	projectRepo *repository.ProjectRepository
	tenantRepo  *repository.TenantRepository
	migration   *MigrationService
	vault       *vault.VaultService
}

// NewOrchestratorService initializes the core BaaS logic.
func NewOrchestratorService(
	p *repository.ProjectRepository,
	t *repository.TenantRepository,
	m *MigrationService,
	v *vault.VaultService,
) *OrchestratorService {
	return &OrchestratorService{
		projectRepo: p,
		tenantRepo:  t,
		migration:   m,
		vault:       v,
	}
}

// ProvisionProject handles full tenant creation: metadata, physical DB, and bootstrap migrations.
func (s *OrchestratorService) ProvisionProject(ctx context.Context, name, slug string) (*domain.Project, error) {
	slog.Info("orchestrator: starting provision", "slug", slug)

	// 1. Sanitize Slug and Generate DB Name.
	cleanSlug := strings.ToLower(strings.ReplaceAll(slug, "-", "_"))
	dbName := fmt.Sprintf("cascata_db_%s", cleanSlug)

	// 2. Generate Credentials (Zero-Trust).
	jwtSecret := s.generateRandomKey(32)
	anonKey := s.generateRandomKey(24)
	serviceKey := s.generateRandomKey(48)

	// 3. Physical Provisioning.
	if err := s.tenantRepo.ProvisionNewDatabase(ctx, dbName); err != nil {
		return nil, fmt.Errorf("service.Orchestrator.Provision: %w", err)
	}

	// 4. Bootstrap Migration (Standard Apartment Template).
	if err := s.migration.BootstrapTenant(ctx, dbName); err != nil {
		slog.Error("bootstrap failed, rolling back physical DB", "dbName", dbName, "error", err)
		_ = s.tenantRepo.DropDatabase(ctx, dbName)
		return nil, fmt.Errorf("service.Orchestrator.Bootstrap: %w", err)
	}

	// 5. Vault Key Storage (K/V-V2)
	vaultPath := fmt.Sprintf("tenant/%s/keys", slug)
	err := s.vault.WriteSecret(ctx, vaultPath, map[string]any{
		"jwt_secret":  jwtSecret,
		"anon_key":    anonKey,
		"service_key": serviceKey,
	})
	if err != nil {
		slog.Error("vault store failed, rolling back physical DB", "dbName", dbName, "error", err)
		_ = s.tenantRepo.DropDatabase(ctx, dbName)
		return nil, fmt.Errorf("service.Orchestrator.Vault: %w", err)
	}

	// 6. Metadata Registration with Transit Encryption.
	// We encrypt the JWTSecret via Vault Transit for a second layer of protection.
	transitKey := "cascata-master-key"
	encryptedJWT, err := s.vault.TransitEncrypt(ctx, transitKey, jwtSecret)
	if err != nil {
		slog.Warn("vault transit failed, using plain secret (check if transit engine is enabled)", "error", err)
		encryptedJWT = jwtSecret
	}

	p := &domain.Project{
		Name:             name,
		Slug:             slug,
		DBName:           dbName,
		JWTSecret:        encryptedJWT,
		AnonKey:          anonKey,
		ServiceKey:       serviceKey,
		LogRetentionDays: 30,
		Metadata:         map[string]interface{}{"provisioned_by": "cascata-go-v1", "vault_path": vaultPath},
	}

	if err := s.projectRepo.Create(ctx, p); err != nil {
		slog.Error("metadata failed, rolling back physical DB", "dbName", dbName, "error", err)
		_ = s.tenantRepo.DropDatabase(ctx, dbName)
		return nil, fmt.Errorf("service.Orchestrator.Register: %w", err)
	}

	slog.Info("project provisioning complete", "slug", slug, "projectId", p.ID)
	return p, nil
}

func (s *OrchestratorService) generateRandomKey(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "insecure_fallback_key" // Should never happen with proper CSPRNG
	}
	return hex.EncodeToString(b)
}
