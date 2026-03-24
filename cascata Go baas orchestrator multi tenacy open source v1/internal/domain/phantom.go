package domain

import "time"

// PhantomFunction represents a serverless edge unit in the Cascata ecosystem.
// It carries the WASM binary and execution metadata for tenant-defined logic.
type PhantomFunction struct {
	ID          string    `json:"id"`
	ProjectSlug string    `json:"project_slug"`
	Name        string    `json:"name"`        // e.g., "process_payment_webhook"
	Description string    `json:"description,omitempty"`
	Binary      []byte    `json:"-"`           // The compiled WASM module
	Runtime     string    `json:"runtime"`     // wasm32-wasi
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FunctionInvocation records a single execution result for audit/billing.
type FunctionInvocation struct {
	FunctionID  string    `json:"function_id"`
	ProjectSlug string    `json:"project_slug"`
	LatencyMS   int64     `json:"latency_ms"`
	Result      string    `json:"result"`      // "success", "error", "timeout"
	Timestamp   time.Time `json:"timestamp"`
}
