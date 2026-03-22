package repository

import (
	"context"
	"fmt"
	"log/slog"

	"cascata/internal/database"
	"cascata/internal/domain"
)

// ProjectRepository handles CRUD for project metadata.
type ProjectRepository struct {
	repo *database.Repository
}

// NewProjectRepository initializes project storage.
func NewProjectRepository(repo *database.Repository) *ProjectRepository {
	return &ProjectRepository{repo: repo}
}

// Create inserts a new project metadata record.
func (r *ProjectRepository) Create(ctx context.Context, p *domain.Project) error {
	const sql = `
		INSERT INTO system.projects (
			name, slug, db_name, jwt_secret, anon_key, service_key, metadata, log_retention_days
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	err := r.repo.Pool.QueryRow(ctx, sql,
		p.Name, p.Slug, p.DBName, p.JWTSecret, p.AnonKey, p.ServiceKey, p.Metadata, p.LogRetentionDays,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return fmt.Errorf("repository.Project.Create: %w", err)
	}

	slog.Info("project record created", "slug", p.Slug, "db", p.DBName)
	return nil
}

// GetByIdentifier retrieves a project by its unique slug or its custom domain.
// Critical for Multi-Tenancy resolution at Phase 2 and 17.
func (r *ProjectRepository) GetByIdentifier(ctx context.Context, identifier string) (*domain.Project, error) {
	const sql = `
		SELECT id, name, slug, db_name, status, custom_domain, default_domain, anon_key, service_key, jwt_secret, metadata, log_retention_days, created_at, updated_at
		FROM system.projects 
		WHERE slug = $1 OR custom_domain = $1 OR default_domain = $1
		LIMIT 1
	`
	p := &domain.Project{}
	err := r.repo.Pool.QueryRow(ctx, sql, identifier).Scan(
		&p.ID, &p.Name, &p.Slug, &p.DBName, &p.Status, &p.CustomDomain, &p.DefaultDomain, &p.AnonKey, &p.ServiceKey, &p.JWTSecret, &p.Metadata, &p.LogRetentionDays, &p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("repository.Project.GetByIdentifier(%s): %w", identifier, err)
	}

	return p, nil
}
