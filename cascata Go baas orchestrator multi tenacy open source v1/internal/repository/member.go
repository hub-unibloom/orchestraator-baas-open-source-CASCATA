package repository

import (
	"context"
	"database/sql"
	"fmt"

	"cascata/internal/database"
	"cascata/internal/domain"
)

// MemberRepository manages members in the `system` schema.
type MemberRepository struct {
	db *database.Repository
}

// NewMemberRepository creates a repository for system operators.
func NewMemberRepository(db *database.Repository) *MemberRepository {
	return &MemberRepository{db: db}
}

// FindByEmail retrieves a member by their primary identifier.
func (r *MemberRepository) FindByEmail(ctx context.Context, email string) (*domain.Member, error) {
	query := `SELECT id, email, password_hash, role, type, mfa_enabled, created_at, updated_at 
	          FROM system.members WHERE email = $1`

	m := &domain.Member{}
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&m.ID, &m.Email, &m.PasswordHash, &m.Role, &m.Type, &m.MFAEnabled, &m.CreatedAt, &m.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("repo.Member: find by email: %w", err)
	}

	return m, nil
}

// Create inserts a new system operator.
func (r *MemberRepository) Create(ctx context.Context, m *domain.Member) error {
	query := `INSERT INTO system.members (email, password_hash, role, type, mfa_enabled)
	          VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`

	err := r.db.Pool.QueryRow(ctx, query, m.Email, m.PasswordHash, m.Role, m.Type, m.MFAEnabled).Scan(
		&m.ID, &m.CreatedAt, &m.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("repo.Member: create: %w", err)
	}

	return nil
}

// Audit logs are now handled via the central AuditService (domain.Auditor).
// The repository focuses exclusively on Member CRUD.
