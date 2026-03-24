package repository

import (
	"context"
	"fmt"

	"cascata/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ResidentRepository handles persistence for Tenant Residents (end-users).
// Implements absolute isolation by operating within a Project's physical pool.
type ResidentRepository struct {
	// We don't store a master pool here because users are tenant-specific.
	// The Pool is provided per-call via the TenantPoolManager in the service layer.
}

func NewResidentRepository() *ResidentRepository {
	return &ResidentRepository{}
}

// FindByIdentifier searches for a resident by Email, CPF or WhatsApp within a tenant context.
func (r *ResidentRepository) FindByIdentifier(ctx context.Context, tx pgx.Tx, identifier string) (*domain.Resident, string, error) {
	const sql = `
		SELECT id, email, cpf, whatsapp, password_hash, role, metadata, created_at, updated_at 
		FROM users 
		WHERE email = $1 OR cpf = $1 OR whatsapp = $1
	`
	var u domain.Resident
	var hashedPassword string

	err := tx.QueryRow(ctx, sql, identifier).Scan(
		&u.ID, &u.Email, &u.CPF, &u.WhatsApp, &hashedPassword, &u.Role, &u.Metadata, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("user.repository.FindByIdentifier: %w", err)
	}

	return &u, hashedPassword, nil
}

// Create persists a new resident within the provided transaction context.
func (r *ResidentRepository) Create(ctx context.Context, tx pgx.Tx, u *domain.Resident, hashedPassword string) error {
	const sql = `
		INSERT INTO users (email, cpf, whatsapp, password_hash, role, metadata) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id, created_at, updated_at
	`
	err := tx.QueryRow(ctx, sql, 
		u.Email, u.CPF, u.WhatsApp, hashedPassword, u.Role, u.Metadata,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("user.repository.Create: %w", err)
	}
	return nil
}

// UpdateLastLogin updates the login timestamp for auditing.
func (r *ResidentRepository) UpdateLastLogin(ctx context.Context, tx pgx.Tx, residentID string) error {
	const sql = `UPDATE users SET last_login = now() WHERE id = $1`
	_, err := tx.Exec(ctx, sql, residentID)
	if err != nil {
		return fmt.Errorf("user.repository.UpdateLastLogin: %w", err)
	}
	return nil
}
