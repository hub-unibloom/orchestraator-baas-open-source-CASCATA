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
	AnonKey             string                 `json:"anon_key"`
	ServiceKey          string                 `json:"service_key"`
	Blocklist           []string               `json:"blocklist"`
	Metadata            map[string]interface{} `json:"metadata"`
	LogRetentionDays    int                    `json:"log_retention_days"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// TenantConfig represents the physical configuration for a tenant database.
type TenantConfig struct {
	Slug   string
	DBName string
	Owner  string
}
