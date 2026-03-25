package domain

import "time"

// AuditEntry is an immutable record of a critical system or database operation (Phase 19).
// It implements a Merkle-chain pattern where each entry includes the hash of the previous one.
type AuditEntry struct {
	ID           string       `json:"id"`
	Project      string       `json:"project"`
	Table        string       `json:"table"`
	Operation    string       `json:"operation"` // e.g., UPDATE, DELETE, POLICY_CHANGE
	IdentityID   string       `json:"identity_id"`
	IdentityType IdentityType `json:"identity_type"`
	Payload      string       `json:"payload"`   // JSON-encoded diff
	Timestamp    time.Time    `json:"timestamp"`
	PrevHash     string       `json:"prev_hash"` // Pointer to the previous entry in the chain
	EntryHash    string       `json:"hash"`      // Calculated hash of this entry (including prev_hash)
	Signature    string       `json:"signature"` // Optional signature from the Native Security key
}

// AuditLedgerDDL is the schema for the immutable audit store (Phase 19).
const AuditLedgerDDL = `
CREATE TABLE IF NOT EXISTS system.audit_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_slug TEXT,
    operation TEXT NOT NULL,
    identity_id TEXT NOT NULL,
    identity_type TEXT NOT NULL,
    table_name TEXT,
    payload JSONB,
    prev_hash TEXT,
    entry_hash TEXT NOT NULL UNIQUE,
    signature TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_identity ON system.audit_ledger(identity_id, identity_type);
CREATE INDEX IF NOT EXISTS idx_audit_project ON system.audit_ledger(project_slug, created_at DESC);
`
