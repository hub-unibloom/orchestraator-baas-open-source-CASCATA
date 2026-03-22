package service

import (
	"context"
	"fmt"
	"log/slog"

	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/repository"
)

// ProjectService manages project lookup and metadata with caching.
type ProjectService struct {
	repo    *repository.ProjectRepository
	pools   *database.TenantPoolManager
}

// NewProjectService initializes the project manager.
func NewProjectService(repo *repository.ProjectRepository, pools *database.TenantPoolManager) *ProjectService {
	return &ProjectService{
		repo:  repo,
		pools: pools,
	}
}

// GetPool retrieves the isolated connection pool for a resolved project.
func (s *ProjectService) GetPool(ctx context.Context, p *domain.Project) (*database.Repository, error) {
	// Tenant specific max connections from metadata, fallback to 10.
	maxConns := 10
	if val, ok := p.Metadata["max_connections"].(float64); ok {
		maxConns = int(val)
	}

	return s.pools.GetPool(ctx, p.Slug, p.DBName, maxConns)
}

// Resolve identifies a project by its unique slug or custom domain.
// Used by the AuthMiddleware to establish project context before CRUD.
func (s *ProjectService) Resolve(ctx context.Context, identifier string) (*domain.Project, error) {
	// L1 Check (Future: In-memory cache for ultra-hot path)
	// L2 Check (Future: Dragonfly cache for cluster-wide consistency)

	// Primary Lookup on Metadata DB.
	p, err := s.repo.GetByIdentifier(ctx, identifier)
	if err != nil {
		slog.Warn("project resolution failed", "identifier", identifier, "error", err)
		return nil, fmt.Errorf("service.Project.Resolve: %w", err)
	}

	return p, nil
}

// UpdateHealth marks project status in the metadata DB.
func (s *ProjectService) UpdateHealth(ctx context.Context, slug string, status string) error {
	// Implementation should update current status.
	slog.Info("project health update", "slug", slug, "status", status)
	return nil
}
