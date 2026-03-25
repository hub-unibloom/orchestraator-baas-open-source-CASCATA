package privacy

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"cascata/internal/domain"
	"cascata/internal/security"
)

// Engine handles data transformation according to the privacy policy (Phase 9 - Native AES).
// It acts as the "Blindagem de Dados" (Shielding), ensuring sensitive information is encrypted in DB
// and only revealed to authorized identities on-the-fly.
type Engine struct {
	security *security.SecurityService
}

func NewEngine(s *security.SecurityService) *Engine {
	return &Engine{security: s}
}

// Regex patterns for PII detection (Phase 24)
var (
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	cpfRegex   = regexp.MustCompile(`\d{3}\.\d{3}\.\d{3}-\d{2}`)
)

// ApplyMasking processes search results to obfuscate or reveal sensitive data.
// It uses the Security Engine to decrypt values if the user has the 'service_role' (Admin) permission.
func (e *Engine) ApplyMasking(ctx context.Context, authCtx *domain.AuthContext, cfg ProjectPrivacyConfig, tableName string, rows []map[string]interface{}) ([]map[string]interface{}, error) {
	if len(rows) == 0 {
		return rows, nil
	}

	tableMasks, hasMasks := cfg.MaskedColumns[tableName]
	if !hasMasks {
		return rows, nil
	}

	// In Cascata, 'service_role' is the only role allowed to transparently see decrypted data (Zero-Trust).
	isAdmin := authCtx.Role == "service_role"

	for i := range rows {
		for col, val := range rows[i] {
			mask, isMasked := tableMasks[col]
			if !isMasked {
				// Auto-Detection of PII (Phase 24)
				if s, ok := val.(string); ok && !isAdmin {
					if emailRegex.MatchString(s) || cpfRegex.MatchString(s) {
						rows[i][col] = "[PII_DETECTED]"
					}
				}
				continue
			}

			// 1. Admin/Service Bypass & Transparent Decryption
			if isAdmin {
				if (mask == HybridEnc || mask == HyperEnc) && val != nil {
					if s, ok := val.(string); ok && security.IsSovereign(s) {
						// On-demand decryption via Security Engine
						dec, err := e.security.Decrypt(ctx, s)
						if err == nil {
							rows[i][col] = dec
						} else {
							rows[i][col] = "[DECRYPT_FAILED]"
						}
					}
				}
				continue
			}

			// 2. Privacy Enforcement for Non-Admins (Residents)
			if val == nil { continue }

			switch mask {
			case Hide:
				rows[i][col] = nil // Completely hidden
			case Mask:
				rows[i][col] = "********"
			case SemiMask:
				s := fmt.Sprintf("%v", val)
				if len(s) > 4 {
					rows[i][col] = s[:4] + "****"
				} else {
					rows[i][col] = "****"
				}
			case Blur:
				rows[i][col] = "[HIDDEN_DATA]"
			case HybridEnc, HyperEnc:
				rows[i][col] = "[ENCRYPTED]"
			}
		}
	}

	return rows, nil
}

// SanitizeWrite strips fields based on column locks or performs transparent encryption.
// Used in POST/PATCH operations to protect PII before it touches the physical disk.
func (e *Engine) SanitizeWrite(ctx context.Context, authCtx *domain.AuthContext, cfg ProjectPrivacyConfig, tableName string, data map[string]interface{}, isUpdate bool) (map[string]interface{}, error) {
	isAdmin := authCtx.Role == "service_role"
	
	// 1. Column Locking (Immutable, Write-once, System-only)
	if tableLocks, ok := cfg.LockedColumns[tableName]; ok {
		for col, lock := range tableLocks {
			if _, exists := data[col]; !exists {
				continue
			}

			switch lock {
			case Immutable:
				delete(data, col) // Never writable via API
			case InsertOnly:
				if isUpdate { delete(data, col) }
			case SystemPut:
				if !isAdmin { delete(data, col) }
			case OtpProtected:
				// Phase 7: Requires active Step-Up session for sensitive field mutation.
				if !isAdmin && !authCtx.HasStepUp {
					slog.Warn("privacy: otp_protected write blocked - no step-up session", "slug", authCtx.ProjectSlug, "col", col)
					delete(data, col)
				}
			}
		}
	}

	// 2. Project-Wide Transparent Encryption (Native AES Bridge)
	// Any column marked as HybridEnc or HyperEnc is automatically encrypted before storage.
	if tableMasks, ok := cfg.MaskedColumns[tableName]; ok {
		for col, mask := range tableMasks {
			val, exists := data[col]
			if !exists || val == nil {
				continue
			}

			if mask == HybridEnc || mask == HyperEnc {
				// We don't re-encrypt if it already looks like a security ciphertext.
				if s, ok := val.(string); ok && security.IsSovereign(s) {
					continue
				}

				// Encrypt via Security Engine using the Master Key.
				enc, err := e.security.Encrypt(ctx, fmt.Sprintf("%v", val))
				if err != nil {
					return nil, fmt.Errorf("privacy: failed to encrypt sensitive column %s: %w", col, err)
				}
				data[col] = enc
			}
		}
	}

	return data, nil
}
