package auth

import (
	"context"
	"fmt"
	"log/slog"

	"cascata/internal/crypto"
	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/repository"
	"github.com/jackc/pgx/v5"
)

// SystemAuthService handles management login for Cascata Members.
// Distinguishes from AuthService which handles Tenant APIs.
type SystemAuthService struct {
	db         *database.Repository
	memberRepo *repository.MemberRepository
	auditSvc   domain.Auditor
}

// NewSystemAuthService initializes the management security layer.
func NewSystemAuthService(db *database.Repository, repo *repository.MemberRepository, audit domain.Auditor) *SystemAuthService {
	return &SystemAuthService{db: db, memberRepo: repo, auditSvc: audit}
}

// AuthenticateMember validates credentials for a system operator (Dashboard login).
func (s *SystemAuthService) AuthenticateMember(ctx context.Context, email, password string) (*domain.Member, error) {
	var member *domain.Member
	var err error

	// 1. Establish Sovereign System Context
	err = s.db.WithRLS(ctx, database.UserClaims{Role: "service_role"}, "cascata", false, func(tx pgx.Tx) error {
		// 1.1 Retrieve member from DB (Isolated by cascata_system)
		member, err = s.memberRepo.FindByEmail(ctx, tx, email)
		if err != nil {
			return fmt.Errorf("system.auth: find member: %w", err)
		}
		if member == nil {
			return fmt.Errorf("system.auth: member not found")
		}

		// 1.2 Log the successful login attempt in Unified Audit Ledger
		_ = s.auditSvc.Log(ctx, tx, "cascata", "MEMBER_LOGIN", member.ID, string(member.Type), map[string]interface{}{"email": email})
		
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 2. Validate Password via Argon2id (Standardized Phase 10)
	isValid, err := crypto.ComparePassword(password, member.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("system.auth: internal error: %w", err)
	}
	
	if !isValid {
		slog.Warn("system.auth: password mismatch", "email", email)
		return nil, fmt.Errorf("system.auth: invalid credentials")
	}

	return member, nil
}

// LogAudit exposes the internal auditor to external handlers without exposing the field.
func (s *SystemAuthService) LogAudit(ctx context.Context, q domain.Queryer, projectSlug, op, actorID, actorType string, payload any) error {
	return s.auditSvc.Log(ctx, q, projectSlug, op, actorID, actorType, payload)
}
