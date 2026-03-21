package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cascata/internal/database"
)

// Interceptor manages the synchronous request/response hooks.
type Interceptor struct {
	repo *database.Repository
}

func NewInterceptor(repo *database.Repository) *Interceptor {
	return &Interceptor{repo: repo}
}

// ForbiddenPatterns contains SQL fragments used for privilege escalation or info leak.
var ForbiddenPatterns = []string{
	"COPY ", "PG_READ_FILE", "PG_LS_DIR", "DO $$", "CAST(", "SLEEP(", "TERMINATE",
}

// ValidateQuery prevents malicious SQL patterns before execution.
func (i *Interceptor) ValidateQuery(sql string) error {
	upper := strings.ToUpper(sql)
	for _, p := range ForbiddenPatterns {
		if strings.Contains(upper, p) {
			return fmt.Errorf("Security Block: SQL Pattern '%s' is forbidden", p)
		}
	}
	return nil
}

// InterceptResponse allows applying logic to the JSON before it is sent to the client.
// Each tenant can have a list of interceptors (Tipo B).
func (i *Interceptor) InterceptResponse(ctx context.Context, projectSlug string, data interface{}) (interface{}, error) {
	// 1. Check for active interceptors in metadata
	// Logic similar to Privacy Engine but for broader transformations.
	
	// TODO: Phase 10 - Execute Automation Nodes for síncrono Interceptor.
	// For Phase 9, we establish the middleware interface.
	
	return data, nil
}
