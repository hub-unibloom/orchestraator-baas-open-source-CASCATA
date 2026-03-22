package vault

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/vault-client-go"
)

// TransitService handles on-the-fly encryption/decryption using Vault.
type TransitService struct {
	client *vault.Client
}

func NewTransitService(client *vault.Client) *TransitService {
	return &TransitService{client: client}
}

// Encrypt wraps the Vault Transit 'encrypt' operation.
func (s *TransitService) Encrypt(ctx context.Context, keyName string, data string) (string, error) {
	resp, err := s.client.Transit.Encrypt(ctx, keyName, vault.TransitEncryptRequest{
		Plaintext: base64.StdEncoding.EncodeToString([]byte(data)),
	})
	if err != nil {
		return "", fmt.Errorf("vault.TransitEncrypt [%s]: %w", keyName, err)
	}

	ciphertext, ok := resp.Data["ciphertext"].(string)
	if !ok {
		return "", fmt.Errorf("vault.TransitEncrypt: ciphertext not found in response")
	}

	return ciphertext, nil
}

// Decrypt wraps the Vault Transit 'decrypt' operation.
func (s *TransitService) Decrypt(ctx context.Context, keyName string, ciphertext string) (string, error) {
	resp, err := s.client.Transit.Decrypt(ctx, keyName, vault.TransitDecryptRequest{
		Ciphertext: ciphertext,
	})
	if err != nil {
		return "", fmt.Errorf("vault.TransitDecrypt [%s]: %w", keyName, err)
	}

	plaintextB64, ok := resp.Data["plaintext"].(string)
	if !ok {
		return "", fmt.Errorf("vault.TransitDecrypt: plaintext not found in response")
	}

	decoded, err := base64.StdEncoding.DecodeString(plaintextB64)
	if err != nil {
		return "", fmt.Errorf("vault.TransitDecrypt.Decode [%s]: %w", keyName, err)
	}

	return string(decoded), nil
}
