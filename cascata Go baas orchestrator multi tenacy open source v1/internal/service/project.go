package service

import (
	"context"
	"fmt"
	"log/slog"

	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/repository"
	"github.com/jackc/pgx/v5"
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
	// 1. Resolve within the Sovereign System Context (slug: cascata)
	// This ensures the search_path is set to cascata_system.
	var p *domain.Project
	err := s.repo.Repo().WithRLS(ctx, database.UserClaims{Role: "service_role"}, "cascata", false, func(tx pgx.Tx) error {
		var err error
		p, err = s.repo.GetByIdentifier(ctx, tx, identifier)
		return err
	})

	if err != nil {
		slog.Warn("project resolution failed", "identifier", identifier, "error", err)
		return nil, fmt.Errorf("service.Project.Resolve: %w", err)
	}

	return p, nil
}

// ListProjects returns all projects registered in the Sovereign System Context.
func (s *ProjectService) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	var projects []*domain.Project
	err := s.repo.Repo().WithRLS(ctx, database.UserClaims{Role: "service_role"}, "cascata", false, func(tx pgx.Tx) error {
		var err error
		projects, err = s.repo.List(ctx, tx)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("service.Project.List: %w", err)
	}

	return projects, nil
}

// UpdateHealth marks project status in the metadata DB.
func (s *ProjectService) UpdateHealth(ctx context.Context, slug string, status string) error {
	// Implementation should update current status.
	slog.Info("project health update", "slug", slug, "status", status)
	return nil
}

// Repo returns the underlying metadata repository. 
func (s *ProjectService) Repo() *repository.ProjectRepository {
	return s.repo
}
