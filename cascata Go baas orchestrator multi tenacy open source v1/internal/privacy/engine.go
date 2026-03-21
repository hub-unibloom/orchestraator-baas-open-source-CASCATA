package privacy

import (
	"context"
	"fmt"
	"strings"

	"cascata/internal/auth"
	"cascata/internal/vault"
)

// Engine handles data transformation according to the privacy policy.
type Engine struct {
	transit *vault.TransitService
}

func NewEngine(t *vault.TransitService) *Engine {
	return &Engine{transit: t}
}

// ApplyMasking processes search results to obfuscate sensitive data.
func (e *Engine) ApplyMasking(ctx context.Context, authCtx auth.AuthContext, cfg ProjectPrivacyConfig, tableName string, rows []map[string]interface{}) ([]map[string]interface{}, error) {
	if cfg.MaskedColumns == nil {
		return rows, nil
	}

	tableMasks, ok := cfg.MaskedColumns[tableName]
	if !ok {
		return rows, nil
	}

	isAdmin := authCtx.Role == "service_role"

	for i := range rows {
		for col, val := range rows[i] {
			mask, ok := tableMasks[col]
			if !ok {
				continue
			}

			// 1. Admin Bypass & Decryption (Project specific key)
			if isAdmin {
				// We only use decryprion on HybridEnc columns.
				if mask == HybridEnc && val != nil {
					// We'll decrypted on demand.
					if s, ok := val.(string); ok && strings.HasPrefix(s, "vault:") {
						dec, err := e.transit.Decrypt(ctx, authCtx.ProjectSlug, s)
						if err == nil {
							rows[i][col] = dec
						}
					}
				}
				continue
			}

			// 2. Privacy Enforcement for Non-Admins
			if val == nil {
				continue
			}

			switch mask {
			case Hide:
				delete(rows[i], col)
			case Mask:
				rows[i][col] = "********"
			case SemiMask:
				s := fmt.Sprintf("%v", val)
				visibleLen := len(s) / 4
				if visibleLen == 0 { visibleLen = 1 }
				rows[i][col] = s[:visibleLen] + strings.Repeat("*", max(3, len(s)-visibleLen))
			case Blur:
				s := fmt.Sprintf("%v", val)
				if len(s) > 5 {
					rows[i][col] = s[:3] + "..." + s[len(s)-2:]
				} else {
					rows[i][col] = "***"
				}
			case HybridEnc, HyperEnc:
				rows[i][col] = "[ENCRYPTED]"
			}
		}
	}

	return rows, nil
}

// SanitizeWrite strips fields based on column locks for INSERT/UPDATE.
func (e *Engine) SanitizeWrite(authCtx auth.AuthContext, cfg ProjectPrivacyConfig, tableName string, data map[string]interface{}, isUpdate bool) (map[string]interface{}, error) {
	if cfg.LockedColumns == nil {
		return data, nil
	}

	tableLocks, ok := cfg.LockedColumns[tableName]
	if !ok {
		return data, nil
	}

	isAdmin := authCtx.Role == "service_role"

	for col, lock := range tableLocks {
		val, exists := data[col]
		if !exists {
			continue
		}

		// Security Locks
		switch lock {
		case Immutable:
			delete(data, col) // Never writable via API
		case InsertOnly:
			if isUpdate {
				delete(data, col) // Only writable on insert
			}
		case SystemPut:
			if !isAdmin {
				delete(data, col) // Only orchestrator can write
			}
		case OtpProtected:
			// TODO: Phase 12 - Step-up Authentication
			// For now, if no step-up claim, block write.
			if !authCtx.HasStepUp {
				delete(data, col)
			}
		}

		// Also perform transparent encryption if it's a HybridEnc column.
		if mask, ok := cfg.MaskedColumns[tableName][col]; ok && mask == HybridEnc && val != nil && !isAdmin {
			// Transparently encrypt before storage.
			enc, err := e.transit.Encrypt(context.Background(), authCtx.ProjectSlug, fmt.Sprintf("%v", val))
			if err == nil {
				data[col] = "vault:" + enc
			}
		}
	}

	return data, nil
}

func max(a, b int) int {
	if a > b { return a }
	return b
}
