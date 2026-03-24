package automation

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"cascata/internal/ai"
	"cascata/internal/communication"
	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/service"
)

// ExecutionContext carries the state, variables and project resolution through the workflow graph.
type ExecutionContext struct {
	Vars        map[string]interface{}
	Payload     *domain.AutomationEvent
	ProjectSlug string
	DBName      string
	MaxConns    int
	PoolManager *database.TenantPoolManager
}

// Broadcaster defines the interface for real-time signaling synergy.
type Broadcaster interface {
	Broadcast(ctx context.Context, slug string, payload interface{}) error
}

// WorkflowEngine acts as the coordinator for tenant-defined automations.
type WorkflowEngine struct {
	repo       *database.Repository
	projectSvc *service.ProjectService
	poolMgr    *database.TenantPoolManager
	postman    communication.Dispatcher
	aiEngine   *ai.Engine
	realtime   Broadcaster
}

func NewWorkflowEngine(repo *database.Repository, projectSvc *service.ProjectService, poolMgr *database.TenantPoolManager, postman communication.Dispatcher, aiEngine *ai.Engine, realtime Broadcaster) *WorkflowEngine {
	return &WorkflowEngine{
		repo:       repo,
		projectSvc: projectSvc,
		poolMgr:    poolMgr,
		postman:    postman,
		aiEngine:   aiEngine,
		realtime:   realtime,
	}
}


// Variable Regex for {{node.field}} or {{trigger.data.email}}
var varRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ResolveVariables replaces {{path}} in a template string using dot-notation lookup.
func ResolveVariables(template string, vars map[string]interface{}) string {
	return varRegex.ReplaceAllStringFunc(template, func(match string) string {
		path := strings.Trim(match[2:len(match)-2], " ")
		val := GetValueByPath(path, vars)
		if val == nil {
			return ""
		}
		return fmt.Sprintf("%v", val)
	})
}

// GetValueByPath retrieves a value from a nested map using dot notation (e.g., "trigger.data.id").
func GetValueByPath(path string, vars map[string]interface{}) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = vars
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok { return nil }
		val, exists := m[part]
		if !exists { return nil }
		current = val
	}
	return current
}

// ResolveObject recursively resolves all string fields within a structure using the vars map.
func (e *WorkflowEngine) ResolveObject(source interface{}, vars map[string]interface{}) interface{} {
	switch v := source.(type) {
	case string:
		return ResolveVariables(v, vars)
	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, val := range v {
			res[k] = e.ResolveObject(val, vars)
		}
		return res
	case []interface{}:
		res := make([]interface{}, len(v))
		for i, val := range v {
			res[i] = e.ResolveObject(val, vars)
		}
		return res
	default:
		return v
	}
}

// ExecuteWorkflow runs a specific workflow graph against an event payload.
// Returns the result of the last node (Phase 9/10 Synchronous Hijacking).
func (e *WorkflowEngine) ExecuteWorkflow(ctx context.Context, wf *domain.Workflow, event *domain.AutomationEvent) (interface{}, error) {
	slog.Info("automation: starting workflow execution", "wf_id", wf.ID, "slug", event.ProjectSlug)

	// 0. Resolve Project Metadata for Pool Management
	proj, err := e.projectSvc.Resolve(ctx, event.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("automation: failed to resolve project meta: %w", err)
	}

	// 0.5 Resolve MaxConns from metadata with fallback
	maxConns := 10
	if val, ok := proj.Metadata["max_connections"].(float64); ok {
		maxConns = int(val)
	}

	// 1. Prepare Initial Execution State
	execCtx := &ExecutionContext{
		Vars: map[string]interface{}{
			"trigger": map[string]interface{}{
				"data":      event.Data,
				"table":     event.Table,
				"operation": event.Operation,
			},
			"$input": event.Data,
		},
		Payload:     event,
		ProjectSlug: event.ProjectSlug,
		DBName:      proj.DBName,
		MaxConns:    maxConns,
		PoolManager: e.poolMgr,
	}

	nodeMap := make(map[string]domain.WorkflowNode)
	for i := range wf.Nodes {
		nodeMap[wf.Nodes[i].ID] = wf.Nodes[i]
	}

	// 2. Execution Loop
	if len(wf.Nodes) == 0 { return nil, nil }
	
	// Start with the first node in the slice (Standard workflow start)
	currentNode := &wf.Nodes[0]
	
	const maxSteps = 100
	for i := 0; i < maxSteps; i++ {
		// execute specific logic for the node with retry strategy (Phase 10 Hardening)
		var output interface{}
		var err error
		
		maxRetries := currentNode.RetryCount
		if maxRetries < 0 { maxRetries = 0 }
		
		for r := 0; r <= maxRetries; r++ {
			output, err = e.ProcessNode(ctx, execCtx, currentNode)
			if err == nil {
				break
			}
			if r < maxRetries {
				slog.Warn("automation: retrying node", "node_id", currentNode.ID, "attempt", r+1, "error", err)
				time.Sleep(time.Duration(r+1) * time.Second) // Simple backoff
			}
		}

		// Map node output to the vars registry
		execCtx.Vars[currentNode.ID] = map[string]interface{}{"data": output}
		execCtx.Vars["$last"] = output

		if err != nil {
			slog.Error("automation: node failure after retries", "node_id", currentNode.ID, "error", err)
			
			if !currentNode.ContinueOnError {
				return nil, fmt.Errorf("node %s: %w", currentNode.ID, err)
			}
			slog.Info("automation: continuing after error (ContinueOnError set)", "node_id", currentNode.ID)
		}

		// Resolve Next Node Link
		if currentNode.NextNodeID == "" {
			break
		}
		
		if next, ok := nodeMap[currentNode.NextNodeID]; ok {
			currentNode = &next
		} else {
			break
		}
	}

	slog.Info("automation: workflow completed successfully", "wf_id", wf.ID)
	return execCtx.Vars["$last"], nil
}
