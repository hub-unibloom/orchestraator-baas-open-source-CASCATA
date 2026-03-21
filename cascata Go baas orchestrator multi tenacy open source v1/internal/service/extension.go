package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cascata/internal/database"
	"cascata/internal/phantom"
)

// ExtensionService handles tenant extension life cycle.
type ExtensionService struct {
	injector *phantom.Injector
	master   *database.Repository // To check global availability
}

// NewExtensionService initializes the extension manager.
func NewExtensionService(repo *database.Repository, injector *phantom.Injector) *ExtensionService {
	return &ExtensionService{
		master:   repo,
		injector: injector,
	}
}

// InstallExtension handles the full injection-link-enable flow.
func (s *ExtensionService) InstallExtension(ctx context.Context, tenantRepo *database.Repository, name string) error {
	slog.Info("extension: installation request", "name", name)

	// 1. Check if it requires Phantom Injection.
	if source, isPhantom := phantom.IsPhantom(name); isPhantom {
		// Check if already available in DB system catalog.
		isReady, _ := s.isAvailableInCatalog(ctx, tenantRepo, name)
		if !isReady {
			// Trigger physical injection.
			if err := s.injector.ExtractFromImage(ctx, source); err != nil {
				return fmt.Errorf("service.Extension.Install: phantom injection: %w", err)
			}

			// Poll catalog for linker detection (Max 30s).
			if err := s.waitForLinker(ctx, tenantRepo, name, 30*time.Second); err != nil {
				return fmt.Errorf("service.Extension.Install: linker timeout: %w", err)
			}
		}
	}

	// 2. Enable in Database (with Cascade).
	// We use direct Exec because CREATE EXTENSION requires superuser or specific role, 
	// which our tenant master connection typically has.
	sql := fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS "%s" CASCADE`, name)
	if _, err := tenantRepo.Pool.Exec(ctx, sql); err != nil {
		return fmt.Errorf("service.Extension.Install: create extension: %w", err)
	}

	slog.Info("extension: enabled successfully", "name", name)
	return nil
}

func (s *ExtensionService) isAvailableInCatalog(ctx context.Context, repo *database.Repository, name string) (bool, error) {
	var exists bool
	sql := `SELECT EXISTS(SELECT 1 FROM pg_available_extensions WHERE name = $1)`
	err := repo.Pool.QueryRow(ctx, sql, name).Scan(&exists)
	return exists, err
}

func (s *ExtensionService) waitForLinker(ctx context.Context, repo *database.Repository, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ready, _ := s.isAvailableInCatalog(ctx, repo, name)
		if ready {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			// poll again
		}
	}
	return fmt.Errorf("timeout waiting for phantom linker to detect extension")
}
