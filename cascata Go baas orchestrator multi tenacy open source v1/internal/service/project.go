package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cascata/internal/domain"
	"cascata/internal/repository"
)

// ProjectService manages project lookup and metadata with caching.
type ProjectService struct {
	repo *repository.ProjectRepository
	// TODO: Add Redis/Dragonfly L2 cache here (Phase 3B)
}

// NewProjectService initializes the project manager.
func NewProjectService(repo *repository.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

// Resolve identifies a project by its unique slug or custom domain.
func (s *ProjectService) Resolve(ctx context.Context, identifier string) (*domain.Project, error) {
	// L1 Check (In-memory cache should be here)
	// L2 Check (Dragonfly cache should be here)

	// Fallback to Metadata DB.
	p, err := s.repo.GetBySlug(ctx, identifier)
	if err != nil {
		// Also try by custom domain (Simplified for now)
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
