package vault

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

// VaultService is the core security pillar for secret management.
type VaultService struct {
	client *vault.Client
}

// NewVaultService establishes a connection to the HashiCorp Vault.
func NewVaultService(addr, token string) (*VaultService, error) {
	client, err := vault.New(
		vault.WithAddress(addr),
		vault.WithRequestTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("vault.New: %w", err)
	}

	if err := client.SetToken(token); err != nil {
		return nil, fmt.Errorf("vault.SetToken: %w", err)
	}

	slog.Info("vault connection established", "addr", addr)
	return &VaultService{client: client}, nil
}

// TransitEncrypt uses the Transit engine to encrypt sensitive data (Resident sovereignity).
func (s *VaultService) TransitEncrypt(ctx context.Context, keyName, plaintext string) (string, error) {
	resp, err := s.client.Secrets.TransitEncrypt(ctx, keyName, schema.TransitEncryptRequest{
		Plaintext: plaintext,
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

// TransitDecrypt reverses the Transit encryption.
func (s *VaultService) TransitDecrypt(ctx context.Context, keyName, ciphertext string) (string, error) {
	resp, err := s.client.Secrets.TransitDecrypt(ctx, keyName, schema.TransitDecryptRequest{
		Ciphertext: ciphertext,
	})
	if err != nil {
		return "", fmt.Errorf("vault.TransitDecrypt [%s]: %w", keyName, err)
	}

	plaintext, ok := resp.Data["plaintext"].(string)
	if !ok {
		return "", fmt.Errorf("vault.TransitDecrypt: plaintext not found in response")
	}

	return plaintext, nil
}

// WriteSecret stores a secret in the KV-V2 engine.
func (s *VaultService) WriteSecret(ctx context.Context, path string, data map[string]any) error {
	_, err := s.client.Secrets.KvV2Write(ctx, path, schema.KvV2WriteRequest{
		Data: data,
	})
	if err != nil {
		return fmt.Errorf("vault.WriteSecret [%s]: %w", path, err)
	}
	return nil
}

// ReadSecret retrieves a secret from the KV-V2 engine.
func (s *VaultService) ReadSecret(ctx context.Context, path string) (map[string]any, error) {
	resp, err := s.client.Secrets.KvV2Read(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("vault.ReadSecret [%s]: %w", path, err)
	}
	return resp.Data.Data, nil
}
