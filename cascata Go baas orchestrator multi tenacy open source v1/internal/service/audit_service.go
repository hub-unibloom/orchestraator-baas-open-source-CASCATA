package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"cascata/internal/database"
	"cascata/internal/domain"
)

// AuditService is the "Immutable Scribe" of Cascata (Phase 19).
// It ensures every critical database operation is signed and chained.
type AuditService struct {
	repo    *database.Repository
	mu      sync.Mutex
	lastHash string
}

func NewAuditService(repo *database.Repository) *AuditService {
	s := &AuditService{repo: repo}
	s.refreshLastHash(context.Background())
	return s
}

func (s *AuditService) refreshLastHash(ctx context.Context) {
	// 1. Fetch the hash of the latest entry in the ledger
	var h string
	// Qualify table with cascata_system schema for global visibility
	err := s.repo.Pool.QueryRow(ctx, "SELECT entry_hash FROM cascata_system.audit_ledger ORDER BY created_at DESC LIMIT 1").Scan(&h)
	if err == nil {
		s.lastHash = h
	} else {
		s.lastHash = "GENESIS_BLOCK_0000"
	}
}

// WriteEntry signs and persists a new audit record into the ledger.
func (s *AuditService) WriteEntry(ctx context.Context, q domain.Queryer, entry *domain.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Link to the previous hash
	entry.PrevHash = s.lastHash

	// 2. Calculate this entry's hash
	content := fmt.Sprintf("%s:%s:%s:%s:%s:%s", entry.PrevHash, entry.Project, entry.Operation, entry.Table, entry.IdentityID, entry.Payload)
	h := sha256.Sum256([]byte(content))
	entry.EntryHash = hex.EncodeToString(h[:])

	// 3. Persist (Blockchain-style immutable append)
	// Sovereign Registration: Persisted within the provided transaction context.
	// Qualified table name ensures visibility across namespaces.
	_, err := q.Exec(ctx, 
		"INSERT INTO cascata_system.audit_ledger (project_slug, operation, identity_id, identity_type, table_name, payload, prev_hash, entry_hash) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		entry.Project, entry.Operation, entry.IdentityID, entry.IdentityType, entry.Table, entry.Payload, entry.PrevHash, entry.EntryHash,
	)
	if err != nil {
		return fmt.Errorf("audit.ledger: failed to write entry: %w", err)
	}

	// 4. Advance the chain
	s.lastHash = entry.EntryHash
	slog.Info("audit.ledger: entry added", "slug", entry.Project, "hash", entry.EntryHash)
	
	return nil
}

// Log is a high-level helper for quick auditing of any system event.
func (s *AuditService) Log(ctx context.Context, q domain.Queryer, projectSlug, op, id, idType string, payload any) error {
	pJson, _ := json.Marshal(payload)
	entry := &domain.AuditEntry{
		Project: projectSlug,
		Operation: op,
		IdentityID: id,
		IdentityType: domain.IdentityType(idType),
		Payload: string(pJson),
	}
	return s.WriteEntry(ctx, q, entry)
}
