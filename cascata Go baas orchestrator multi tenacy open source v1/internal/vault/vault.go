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
	token  string
}

// NewVaultService establishes a connection and starts the vital renewal heartbeat.
func NewVaultService(addr, token string) (*VaultService, error) {
	client, err := vault.New(
		vault.WithAddress(addr),
		vault.WithRequestTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("vault.New: %w", err)
	}

	if err := client.SetToken(token); err != nil {
		return nil, fmt.Errorf("vault.SetToken: %w", err)
	}

	svc := &VaultService{
		client: client,
		token:  token,
	}

	// REALITY CHECK: Start Life-Pulse Renewal System
	go svc.keepTokenAlive(context.Background())

	slog.Info("vault synergy established: auto-renewal pulse active", "addr", addr)
	return svc, nil
}

// keepTokenAlive ensures the orchestrator never loses its keys due to TTL expiration.
func (s *VaultService) keepTokenAlive(ctx context.Context) {
	ticker := time.NewTicker(12 * time.Hour) // Pulse every 12h for conservative safety
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Synergic Renewal Pulse
			_, err := s.client.Auth.TokenRenewSelf(ctx, schema.TokenRenewSelfRequest{})
			if err != nil {
				slog.Error("vault: renewal pulse failure - security integrity at risk", "error", err)
				continue
			}
			slog.Debug("vault: identity lease extended successfully")
		}
	}
}

// GetClient returns the underlying vault client for specialized engine access.
func (s *VaultService) GetClient() *vault.Client {
	return s.client
}

// WriteSecret stores a secret in the KV-V2 engine using explicit mount pathing with retry logic.
func (s *VaultService) WriteSecret(ctx context.Context, mount, path string, data map[string]any) error {
	var err error
	for i := 0; i < 3; i++ {
		_, err = s.client.Secrets.KvV2Write(ctx, path, schema.KvV2WriteRequest{
			Data: data,
		}, vault.WithMountPath(mount))
		
		if err == nil {
			return nil
		}
		slog.Warn("vault: write attempt failed, retrying...", "mount", mount, "path", path, "attempt", i+1, "error", err)
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	
	slog.Error("vault: write terminal failure", "mount", mount, "path", path, "error", err)
	return fmt.Errorf("vault.WriteSecret [%s/%s]: %w", mount, path, err)
}

// ReadSecret retrieves a secret from the KV-V2 engine with retry logic.
func (s *VaultService) ReadSecret(ctx context.Context, mount, path string) (map[string]any, error) {
	var err error
	var resp *vault.Response[schema.KvV2ReadResponse]

	for i := 0; i < 3; i++ {
		resp, err = s.client.Secrets.KvV2Read(ctx, path, vault.WithMountPath(mount))
		if err == nil {
			if resp.Data.Data == nil {
				return nil, fmt.Errorf("vault.ReadSecret: no data payload at [%s/%s]", mount, path)
			}
			return resp.Data.Data, nil
		}
		slog.Warn("vault: read attempt failed, retrying...", "mount", mount, "path", path, "attempt", i+1, "error", err)
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}

	slog.Error("vault: read terminal failure", "mount", mount, "path", path, "error", err)
	return nil, fmt.Errorf("vault.ReadSecret [%s/%s]: %w", mount, path, err)
}

// ProvisionTenantVault creates the isolated secret infrastructure for a new project.
func (s *VaultService) ProvisionTenantVault(ctx context.Context, slug string) error {
	slog.Info("vault: initiating sovereign genesis for tenant", "slug", slug)
	
	mountPath := "cascata"
	secretPath := fmt.Sprintf("tenants/%s/config", slug)
	
	data := map[string]any{
		"genesis_at":    time.Now().Format(time.RFC3339),
		"status":        "sovereign_active",
		"isolation_v":   "1.0.0.0",
		"engine":        "sovereign_baas",
	}

	return s.WriteSecret(ctx, mountPath, secretPath, data)
}
