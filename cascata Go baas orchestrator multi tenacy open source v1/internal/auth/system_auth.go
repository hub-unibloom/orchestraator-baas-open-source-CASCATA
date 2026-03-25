package auth

import (
	"context"
	"fmt"
	"log/slog"

	"cascata/internal/crypto"
	"cascata/internal/domain"
	"cascata/internal/repository"
)

// SystemAuthService handles management login for Cascata Members.
// Distinguishes from AuthService which handles Tenant APIs.
type SystemAuthService struct {
	memberRepo *repository.MemberRepository
	auditSvc   domain.Auditor
}

// NewSystemAuthService initializes the management security layer.
func NewSystemAuthService(repo *repository.MemberRepository, audit domain.Auditor) *SystemAuthService {
	return &SystemAuthService{memberRepo: repo, auditSvc: audit}
}

// AuthenticateMember validates credentials for a system operator (Dashboard login).
func (s *SystemAuthService) AuthenticateMember(ctx context.Context, email, password string) (*domain.Member, error) {
	// 1. Retrieve member from DB
	member, err := s.memberRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("system.auth: find member: %w", err)
	}
	if member == nil {
		return nil, fmt.Errorf("system.auth: member not found")
	}

	// 2. Validate Password via Argon2id (Standardized Phase 10)
	isValid, err := crypto.ComparePassword(password, member.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("system.auth: internal error: %w", err)
	}
	
	if !isValid {
		slog.Warn("system.auth: password mismatch", "email", email, "hash_prefix", member.PasswordHash[:15])
		return nil, fmt.Errorf("system.auth: invalid credentials")
	}

	// 3. Log the successful login attempt in Unified Audit Ledger
	_ = s.auditSvc.Log(ctx, "", "MEMBER_LOGIN", member.ID, string(member.Type), map[string]interface{}{"email": email})

	return member, nil
}

// LogAudit exposes the internal auditor to external handlers without exposing the field.
func (s *SystemAuthService) LogAudit(ctx context.Context, projectSlug, op, actorID, actorType string, payload any) error {
	return s.auditSvc.Log(ctx, projectSlug, op, actorID, actorType, payload)
}
