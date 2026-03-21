package mcp

import "time"

// ToolCall represents an AI-triggered action via MCP.
type ToolCall struct {
	ID            string    `json:"id"`
	AgentID       string    `json:"agent_id"`
	ToolName      string    `json:"tool_name"`
	TargetURL     string    `json:"target_url,omitempty"`
	RowsRequested int       `json:"rows_requested"`
	Status        string    `json:"status"`
	Metadata      map[string]interface{} `json:"metadata"`
	Timestamp     time.Time `json:"timestamp"`
}
