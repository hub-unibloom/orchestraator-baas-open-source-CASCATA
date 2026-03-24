package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

// OTPManager handles generation, storage and verification of ephemeral codes.
// Utilizes Dragonfly for high-performance, self-expiring security tokens (Phase 10.1).
type OTPManager struct {
	dfly *redis.Client
}

func NewOTPManager(dfly *redis.Client) *OTPManager {
	return &OTPManager{dfly: dfly}
}

// IssueOTP generates a 6-digit code and persists it in Dragonfly with a 5-minute TTL.
func (m *OTPManager) IssueOTP(ctx context.Context, projectSlug, identifier string) (string, error) {
	// Generate 6-digit numeric code
	code, err := m.generateCode(6)
	if err != nil {
		return "", fmt.Errorf("otp: generation failed: %w", err)
	}

	key := fmt.Sprintf("cascata:otp:%s:%s", projectSlug, identifier)
	
	// Store with TTL (5 minutes as per standard security)
	err = m.dfly.Set(ctx, key, code, 5*time.Minute).Err()
	if err != nil {
		return "", fmt.Errorf("otp: persistence in dragonfly failed: %w", err)
	}

	return code, nil
}

// VerifyOTP checks if the provided code matches the one stored in Dragonfly.
func (m *OTPManager) VerifyOTP(ctx context.Context, projectSlug, identifier, code string) (bool, error) {
	key := fmt.Sprintf("cascata:otp:%s:%s", projectSlug, identifier)
	
	val, err := m.dfly.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil // Expired or not found
	}
	if err != nil {
		return false, err
	}

	if val == code {
		// Validated: Remove from Dragonfly to prevent reuse
		m.dfly.Del(ctx, key)
		return true, nil
	}

	return false, nil
}

func (m *OTPManager) generateCode(length int) (string, error) {
	const charset = "0123456789"
	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}
	return string(result), nil
}
