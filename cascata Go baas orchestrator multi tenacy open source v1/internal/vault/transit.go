package vault

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
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
	resp, err := s.client.Secrets.TransitEncrypt(ctx, keyName, schema.TransitEncryptRequest{
		Plaintext: base64.StdEncoding.EncodeToString([]byte(data)),
	})
	if err != nil {
		return "", fmt.Errorf("vault.TransitEncrypt [%s]: %w", keyName, err)
	}

	return resp.Data.Ciphertext, nil
}

// Decrypt wraps the Vault Transit 'decrypt' operation.
func (s *TransitService) Decrypt(ctx context.Context, keyName string, ciphertext string) (string, error) {
	resp, err := s.client.Secrets.TransitDecrypt(ctx, keyName, schema.TransitDecryptRequest{
		Ciphertext: ciphertext,
	})
	if err != nil {
		return "", fmt.Errorf("vault.TransitDecrypt [%s]: %w", keyName, err)
	}

	decoded, err := base64.StdEncoding.DecodeString(resp.Data.Plaintext)
	if err != nil {
		return "", fmt.Errorf("vault.TransitDecrypt.Decode [%s]: %w", keyName, err)
	}

	return string(decoded), nil
}
