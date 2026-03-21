package automation

import (
	"context"
	"fmt"
	"strings"

	"cascata/internal/database"
	"cascata/internal/vault"
)

// EnvManager handles public and secret environment variables for projects.
type EnvManager struct {
	repo    *database.Repository
	transit *vault.TransitService
}

func NewEnvManager(repo *database.Repository, transit *vault.TransitService) *EnvManager {
	return &EnvManager{
		repo:    repo,
		transit: transit,
	}
}

// ResolveSecret resolves a Vault URI (vault://KEY) safely in memory.
func (m *EnvManager) ResolveSecret(ctx context.Context, projectSlug, uri string) (string, error) {
	if !strings.HasPrefix(uri, "vault://") {
		return uri, nil
	}
	
	key := strings.TrimPrefix(uri, "vault://")
	
	// Transit-based secrets or vault secret engine? 
	// As per Phase 4/7: ProjectSlug is the key namespace in transit.
	decrypted, err := m.transit.Decrypt(ctx, projectSlug, key)
	if err != nil {
		return "", fmt.Errorf("env.ResolveSecret: %w", err)
	}
	
	return decrypted, nil
}

// FetchProjectEnvs loads all environment variables for a project.
func (m *EnvManager) FetchProjectEnvs(ctx context.Context, projectSlug string) (map[string]interface{}, error) {
	sql := `SELECT key, value, is_secret FROM system.project_envs WHERE project_slug = $1`
	rows, err := m.repo.Pool.Query(ctx, sql, projectSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	envs := make(map[string]interface{})
	for rows.Next() {
		var key, val string
		var secret bool
		if err := rows.Scan(&key, &val, &secret); err != nil {
			return nil, err
		}
		
		// If secret, we only return the value if requested inside the engine
		// For the context, we just mark it as resolved on demand.
		envs[key] = val
	}
	return envs, nil
}
