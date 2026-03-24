package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateRandomKey creates a cryptographically secure random hex string of the given length.
func GenerateRandomKey(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("crypto.GenerateRandomKey: entropy exhaustion: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateProjectCredentials creates the identity trio for a new tenant: Anon, Service and JWT Secret.
func GenerateProjectCredentials() (anon, service, jwtSecret string, err error) {
	anon, err = GenerateRandomKey(32)
	if err != nil { return }
	
	service, err = GenerateRandomKey(48) // Service keys are longer for higher entropy
	if err != nil { return }
	
	jwtSecret, err = GenerateRandomKey(64) // JWT Secret standard 256-bit (64 hex chars)
	if err != nil { return }
	
	return
}
