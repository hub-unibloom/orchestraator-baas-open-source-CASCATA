package domain

import (
	"time"
)

// WorkflowTrigger represents the condition that initiates an automation.
type WorkflowTrigger struct {
	Table     string   `json:"table"`
	Events    []string `json:"events"` // INSERT, UPDATE, DELETE
	Condition string   `json:"condition,omitempty"` // Optional JS expression
}

// NodeType defines the specialized action performed by a workflow step.
type NodeType string

const (
	NodeHTTP   NodeType = "http_request"
	NodeScript NodeType = "js_script"
	NodeEmail  NodeType = "send_email"
	NodeVault  NodeType = "vault_access"
	NodeDB     NodeType = "db_operation"
)

// WorkflowNode is an individual step in the automation graph.
type WorkflowNode struct {
	ID              string                 `json:"id"`
	Type            NodeType               `json:"type"`
	Config          map[string]interface{} `json:"config"`
	ContinueOnError bool                   `json:"continue_on_error,omitempty"` // Phase 10 Hardening
	RetryCount      int                    `json:"retry_count,omitempty"`      // Phase 10 Hardening
	NextNodeID      string                 `json:"next_node_id,omitempty"`
}

// Workflow is the top-level container for a tenant's automation logic.
type Workflow struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string                 `json:"type"` // "AUTOMATION", "API_INTERCEPT", "WEBHOOK"
	Config      map[string]interface{} `json:"config"` // Metadata for trigger conditions
	Trigger     WorkflowTrigger        `json:"trigger"`
	Nodes       []WorkflowNode         `json:"nodes"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
}

// AutomationEvent carries the payload of a database change to the workflow engine.
type AutomationEvent struct {
	ProjectSlug string                 `json:"project_slug"`
	Table       string                 `json:"table"`
	Operation   string                 `json:"operation"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
}
