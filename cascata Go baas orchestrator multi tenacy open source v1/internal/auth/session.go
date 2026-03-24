package auth

import (
	"context"
	"fmt"
	"time"

	"cascata/internal/domain"
	"github.com/golang-jwt/jwt/v5" 
)

// SessionManager handles issuance of efemeral management sessions.
type SessionManager struct {
	jwtSecret []byte
}

// NewSessionManager initializes session logic for dashboard members.
func NewSessionManager(secret string) *SessionManager {
	return &SessionManager{jwtSecret: []byte(secret)}
}

// IssueSessionToken generates a signed JWT for a system operator.
// Valid for 15 minutes as per Phase 3 Philosophy.
func (s *SessionManager) IssueSessionToken(ctx context.Context, member *domain.Member) (string, error) {
	claims := jwt.MapClaims{
		"sub":   member.ID,
		"email": member.Email,
		"role":  string(member.Role),
		"type":  string(member.Type),
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateSession ensures the dashboard request came from a valid operator.
func (s *SessionManager) ValidateSession(ctx context.Context, tokenString string) (*domain.AuthContext, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("session: invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("session: invalid claims")
	}

	return &domain.AuthContext{
		Mode:         domain.ModeService,
		IdentityType: domain.IdentityCascataMember,
		Role:         fmt.Sprintf("%v", claims["role"]),
		UserID:       fmt.Sprintf("%v", claims["sub"]),
		Email:        fmt.Sprintf("%v", claims["email"]),
	}, nil
}
