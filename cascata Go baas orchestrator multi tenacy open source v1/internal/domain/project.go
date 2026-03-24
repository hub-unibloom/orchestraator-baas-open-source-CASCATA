package domain

import (
	"time"
)

// Project represents a tenant's metadata within the orchestrator.
type Project struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	Slug                string                 `json:"slug"`
	Status              string                 `json:"status"`
	DBName              string                 `json:"db_name"`
	CustomDomain        *string                `json:"custom_domain,omitempty"`
	DefaultDomain       *string                `json:"default_domain,omitempty"`
	SSLCertificateSource *string               `json:"ssl_certificate_source,omitempty"`
	JWTSecret           string                 `json:"-"`
	SecondarySecretHash string                 `json:"-"` // Extra identity layer (Agency/Multi-User)
	AnonKey             string                 `json:"anon_key"`
	ServiceKey          string                 `json:"service_key"`
	PreviousServiceKey  string                 `json:"-"` 
	
	// Regionalization & Timing
	Region              string                 `json:"region"`
	TimeZone            string                 `json:"timezone"`

	// Resource Quotas (Phase 24: High Density Hardening)
	MaxUsers            int                    `json:"max_users"`
	MaxConns            int                    `json:"max_conns"`
	MaxStorageMB        int64                  `json:"max_storage_mb"`
	MaxDBWeightMB       int64                  `json:"max_db_weight_mb"`

	RolloverAt          *time.Time             `json:"rollover_at,omitempty"`
	Blocklist           []string               `json:"blocklist"`
	Metadata            map[string]interface{} `json:"metadata"`
	LogRetentionDays    int                    `json:"log_retention_days"`
	BackupRetentionDays int                    `json:"backup_retention_days"`
	LastBackupAt        *time.Time             `json:"last_backup_at,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// TenantConfig represents the physical configuration for a tenant database.
type TenantConfig struct {
	Slug   string
	DBName string
	Owner  string
}
