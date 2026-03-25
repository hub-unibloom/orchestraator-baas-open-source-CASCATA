package automation

import (
	"context"
	"fmt"
	"strings"

	"cascata/internal/database"
	"cascata/internal/security"
)

// EnvManager handles public and secret environment variables for projects.
type EnvManager struct {
	repo     *database.Repository
	security *security.SecurityService
}

func NewEnvManager(repo *database.Repository, security *security.SecurityService) *EnvManager {
	return &EnvManager{
		repo:     repo,
		security: security,
	}
}

// ResolveSecret resolves an encrypted URI (cascata://ENCRYPTED_DATA) safely in memory.
func (m *EnvManager) ResolveSecret(ctx context.Context, projectSlug, uri string) (string, error) {
	// 1. If it's already a full security-prefixed string, use it directly.
	if strings.HasPrefix(uri, "cascata:") {
		return m.security.Decrypt(ctx, uri)
	}

	// 2. If it's a URI-style pointer (cascata://ENCRYPTED_DATA), extract the payload.
	if strings.HasPrefix(uri, "cascata://") {
		payload := strings.TrimPrefix(uri, "cascata://")
		// Ensure the payload has the required prefix for the security engine.
		if !strings.HasPrefix(payload, "cascata:") {
			payload = "cascata:" + payload
		}
		return m.security.Decrypt(ctx, payload)
	}
	return uri, nil
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
