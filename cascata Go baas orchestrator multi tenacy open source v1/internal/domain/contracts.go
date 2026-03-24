package domain

import (
	"context"
)

// Auditor defines the contract for immutable logging and system state tracking (Phase 19).
// It decouples the Database layer from the Service layer.
type Auditor interface {
	WriteEntry(ctx context.Context, entry *AuditEntry) error
	Log(ctx context.Context, projectSlug, op, actorID, actorType string, payload any) error
}

// ProjectResolver handles tenant metadata lookup without requiring a circular link to the ProjectService (Phase 2).
type ProjectResolver interface {
	Resolve(ctx context.Context, slug string) (*Project, error)
}

// AIInquisitor defines the brain's intent analysis for the SQL interceptors (Phase 28).
type AIInquisitor interface {
	AnalyzeIntent(ctx context.Context, projectSlug, sql string) (any, error)
}

// PhantomGuardian manages technical injections and WASM logic execution (Phase 14).
type PhantomGuardian interface {
	InvokeFunction(ctx context.Context, projectSlug, functionName string, binary []byte, params map[string]interface{}) ([]byte, error)
}
