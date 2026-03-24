package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"cascata/internal/database"
	"cascata/internal/domain"
)

// ProjectRepository handles CRUD for project metadata.
// It acts as the "Oracle" of the Cascata Architecture, storing all tenant routes and secrets.
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
			name, slug, db_name, jwt_secret, anon_key, service_key, metadata, log_retention_days, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	err := r.repo.Pool.QueryRow(ctx, sql,
		p.Name, p.Slug, p.DBName, p.JWTSecret, p.AnonKey, p.ServiceKey, p.Metadata, p.LogRetentionDays, p.Status,
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

// List all registered projects.
// Used exclusively by the Cascata Dashboard in Management Context.
func (r *ProjectRepository) List(ctx context.Context) ([]*domain.Project, error) {
	const sql = `
		SELECT id, name, slug, db_name, status, created_at, updated_at 
		FROM system.projects 
		ORDER BY created_at DESC
	`
	rows, err := r.repo.Pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("repository.Project.List: %w", err)
	}
	defer rows.Close()

	var projects []*domain.Project
	for rows.Next() {
		p := &domain.Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.DBName, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}
// GetWorkflows returns all automation graphs for a project from the metadata storage (Phase 10).
func (r *ProjectRepository) GetWorkflows(ctx context.Context, slug string) ([]domain.Workflow, error) {
	const sql = "SELECT metadata->'workflows' FROM system.projects WHERE slug = $1"
	var raw json.RawMessage
	err := r.repo.Pool.QueryRow(ctx, sql, slug).Scan(&raw)
	if err != nil { return nil, err }

	var wfs []domain.Workflow
	if len(raw) > 0 && string(raw) != "null" {
		_ = json.Unmarshal(raw, &wfs)
	}
	return wfs, nil
}
// Delete removes a project from the metadata storage.
func (r *ProjectRepository) Delete(ctx context.Context, slug string) error {
	const sql = "DELETE FROM system.projects WHERE slug = $1"
	_, err := r.repo.Pool.Exec(ctx, sql, slug)
	if err != nil {
		return fmt.Errorf("repository.Project.Delete: %w", err)
	}
	slog.Warn("project metadata PURGED", "slug", slug)
	return nil
}
