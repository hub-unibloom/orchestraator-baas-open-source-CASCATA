package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// SecurityService provides the sovereign, native cryptographic core for Cascata.
// It uses AES-256-GCM to protect all sensitive material (JWT secrets, API keys, etc)
// anchored to a high-entropy Master Key.
type SecurityService struct {
	masterKey []byte
}

// NewSecurityService creates a new instance using a 32-byte master key.
func NewSecurityService(masterKey string) (*SecurityService, error) {
	keyBytes := []byte(masterKey)
	if len(keyBytes) < 32 {
		return nil, fmt.Errorf("security: master key must be at least 32 characters long")
	}
	return &SecurityService{masterKey: keyBytes[:32]}, nil
}

// Encrypt performs AES-GCM encryption and returns a 'cascata:' prefixed base64 string.
func (s *SecurityService) Encrypt(ctx context.Context, plaintext string) (string, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return "", fmt.Errorf("security.Encrypt: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("security.Encrypt: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("security.Encrypt: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return "cascata:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt handles 'cascata:' prefixed strings and returns the original plaintext.
func (s *SecurityService) Decrypt(ctx context.Context, data string) (string, error) {
	if !strings.HasPrefix(data, "cascata:") {
		return "", fmt.Errorf("security.Decrypt: missing sovereign prefix")
	}

	raw := strings.TrimPrefix(data, "cascata:")
	ciphertext, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("security.Decrypt: decode failed: %w", err)
	}

	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("security.Decrypt: data too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", fmt.Errorf("security.Decrypt: decryption failed (verify Master Key): %w", err)
	}

	return string(plaintext), nil
}

// IsSovereign returns true if the data is protected by the Cascata engine.
func IsSovereign(data string) bool {
	return strings.HasPrefix(data, "cascata:")
}
}
