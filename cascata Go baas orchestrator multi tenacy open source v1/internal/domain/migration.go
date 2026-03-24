package domain

import "time"

// Migration statuses
const (
	MigrationPending   = "PENDING"
	MigrationRunning   = "RUNNING"
	MigrationCompleted = "COMPLETED"
	MigrationFailed    = "FAILED"
)

// Migration represents a single schema structural change.
type Migration struct {
	ID        string    `json:"id"`        // Unique UUID for the migration job
	Slug      string    `json:"slug"`      // Project slug (Tenant identification)
	Name      string    `json:"name"`      // Unique name, e.g., "001_add_user_bio"
	DDL       string    `json:"ddl"`       // SQL content (Raw audit)
	Checksum  string    `json:"checksum"`  // SHA256 of DDL
	Status    string    `json:"status"`    // PENDING, RUNNING, COMPLETED, FAILED
	Error     string    `json:"error,omitempty"`
	AppliedAt time.Time `json:"applied_at"`
}

// MigrationTableDDL is the internal schema required in every tenant DB to track history (Phase 15).
const MigrationTableDDL = `
CREATE TABLE IF NOT EXISTS cascata_migrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL,
    name TEXT UNIQUE NOT NULL,
    ddl TEXT NOT NULL,
    checksum TEXT NOT NULL,
    status TEXT NOT NULL,
    error TEXT,
    applied_at TIMESTAMPTZ DEFAULT NOW()
);
`
