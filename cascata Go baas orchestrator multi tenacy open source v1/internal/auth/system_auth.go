package auth

import (
	"context"
	"fmt"
	"log/slog"

	"cascata/internal/domain"
	"cascata/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// SystemAuthService handles management login for Cascata Members.
// Distinguishes from AuthService which handles Tenant APIs.
type SystemAuthService struct {
	memberRepo *repository.MemberRepository
}

// NewSystemAuthService initializes the management security layer.
func NewSystemAuthService(repo *repository.MemberRepository) *SystemAuthService {
	return &SystemAuthService{memberRepo: repo}
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

	// 2. Validate Password via Bcrypt (Matches Cost 12+ configured at DB)
	if err := bcrypt.CompareHashAndPassword([]byte(member.PasswordHash), []byte(password)); err != nil {
		slog.Warn("system.auth: invalid password for member", "email", email)
		return nil, fmt.Errorf("system.auth: invalid credentials")
	}

	// 3. Log the successful login attempt in Audit Trail
	_ = s.memberRepo.LogActivity(ctx, &domain.AuditLog{
		MemberID:   member.ID,
		MemberType: member.Type,
		Action:     "MEMBER_LOGIN",
		EntityType: "system.members",
		EntityID:   member.ID,
		Metadata:   map[string]interface{}{"email": email},
	})

	return member, nil
}

// HashPassword generates a secure Bcrypt hash from a plaintext string.
// Used for member creation during install.sh via local Go binary.
func HashPassword(password string) (string, error) {
	// Cost 12 is the minimum production standard for managing infra.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("auth.Hash: calculation failed: %w", err)
	}
	return string(hash), nil
}
