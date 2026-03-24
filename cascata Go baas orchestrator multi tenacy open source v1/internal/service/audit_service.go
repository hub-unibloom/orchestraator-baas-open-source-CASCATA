package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	err := s.repo.Pool.QueryRow(ctx, "SELECT entry_hash FROM system.audit_ledger ORDER BY created_at DESC LIMIT 1").Scan(&h)
	if err == nil {
		s.lastHash = h
	} else {
		s.lastHash = "GENESIS_BLOCK_0000"
	}
}

// WriteEntry signs and persists a new audit record into the ledger.
func (s *AuditService) WriteEntry(ctx context.Context, entry *domain.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Link to the previous hash
	entry.PrevHash = s.lastHash

	// 2. Calculate this entry's hash
	content := fmt.Sprintf("%s:%s:%s:%s:%s:%s", entry.PrevHash, entry.Project, entry.Operation, entry.Table, entry.ActorID, entry.Payload)
	h := sha256.Sum256([]byte(content))
	entry.EntryHash = hex.EncodeToString(h[:])

	// 3. Persist (Blockchain-style immutable append)
	_, err := s.repo.Pool.Exec(ctx, 
		"INSERT INTO system.audit_ledger (project_slug, operation, actor_id, actor_type, table_name, payload, prev_hash, entry_hash) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		entry.Project, entry.Operation, entry.ActorID, entry.ActorType, entry.Table, entry.Payload, entry.PrevHash, entry.EntryHash,
	)
	if err != nil {
		return fmt.Errorf("audit.ledger: failed to write entry: %w", err)
	}

	// 4. Advance the chain
	s.lastHash = entry.EntryHash
	slog.Info("audit.ledger: entry added", "slug", entry.Project, "hash", entry.EntryHash)
	
	return nil
}
