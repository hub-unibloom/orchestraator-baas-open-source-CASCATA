package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cascata/internal/auth"
	"cascata/internal/database"
)

// Context carries the execution state through the graph.
type Context struct {
	Vars        map[string]interface{}
	Payload     interface{}
	ProjectSlug string
	AuthCtx     auth.AuthContext
	Repo        *database.Repository
	ProjectPool *database.TenantPool
}

// Node represents a single unit of logic in the automation graph.
type Node struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
	Next   interface{}            `json:"next"` // Can be string, []string or map[string]string
}

// Engine orchestrates the execution of Node sequences.
type Engine struct {
	repo *database.Repository
}

func NewEngine(repo *database.Repository) *Engine {
	return &Engine{repo: repo}
}

// Variable Regex for {{node.field}} or {{trigger.payload}}
var varRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ResolveVariables replaces {{path}} in a template string with values from vars map.
func ResolveVariables(template string, vars map[string]interface{}) string {
	return varRegex.ReplaceAllStringFunc(template, func(match string) string {
		path := strings.Trim(match[2:len(match)-2], " ")
		val := GetVar(path, vars)
		if val == nil {
			return ""
		}
		return fmt.Sprintf("%v", val)
	})
}

// GetVar retrieves a value from a nested map using dot notation.
func GetVar(path string, vars map[string]interface{}) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = vars
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		val, exists := m[part]
		if !exists {
			return nil
		}
		current = val
	}
	return current
}

// ResolveObject recursively resolves all strings inside a map/slice using vars.
func ResolveObject(source interface{}, vars map[string]interface{}) interface{} {
	switch v := source.(type) {
	case string:
		return ResolveVariables(v, vars)
	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, val := range v {
			res[k] = ResolveObject(val, vars)
		}
		return res
	case []interface{}:
		res := make([]interface{}, len(v))
		for i, val := range v {
			res[i] = ResolveObject(val, vars)
		}
		return res
	default:
		return v
	}
}

// Run executes a graph of nodes for a given payload and context.
func (e *Engine) Run(ctx context.Context, nodes []Node, payload interface{}, automationID string, projectSlug string) (interface{}, error) {
	// 1. Initial State
	execCtx := &Context{
		Vars: map[string]interface{}{
			"trigger": map[string]interface{}{
				"payload": payload,
			},
			"$input": payload,
		},
		Payload:     payload,
		ProjectSlug: projectSlug,
	}

	nodeMap := make(map[string]Node)
	var startNode *Node
	for i := range nodes {
		nodeMap[nodes[i].ID] = nodes[i]
		if nodes[i].Type == "trigger" {
			startNode = &nodes[i]
		}
	}

	if startNode == nil {
		return nil, fmt.Errorf("no trigger node found")
	}

	// 2. Execution Loop (Safety limit of 100 steps)
	currentNode := startNode
	for i := 0; i < 100; i++ {
		res, err := e.ProcessNode(ctx, execCtx, currentNode)
		
		// Map output to vars
		execCtx.Vars[currentNode.ID] = map[string]interface{}{"data": res}
		
		if err != nil {
			// Handle failure path
			nextID := e.getOnFailure(currentNode)
			if nextID == "" {
				return nil, fmt.Errorf("node %s failed: %w", currentNode.ID, err)
			}
			if n, ok := nodeMap[nextID]; ok {
				currentNode = &n
				continue
			}
			return nil, err
		}

		// Resolve Next Node
		nextID := e.getOnSuccess(currentNode, res)
		if nextID == "" {
			break
		}
		
		if n, ok := nodeMap[nextID]; ok {
			currentNode = &n
		} else {
			break
		}
	}

	return execCtx.Vars["$output"], nil
}

func (e *Engine) getOnSuccess(node *Node, result interface{}) string {
	switch v := node.Next.(type) {
	case string:
		return v
	case []string:
		if len(v) > 0 { return v[0] }
	case map[string]interface{}:
		if node.Type == "conditional" {
			resBool, _ := result.(bool)
			if resBool {
				if t, ok := v["true"].(string); ok { return t }
			} else {
				if f, ok := v["false"].(string); ok { return f }
			}
		}
		if s, ok := v["out"].(string); ok { return s }
		if s, ok := v["next"].(string); ok { return s }
	}
	return ""
}

func (e *Engine) getOnFailure(node *Node) string {
	if m, ok := node.Next.(map[string]interface{}); ok {
		if f, ok := m["error"].(string); ok { return f }
		if f, ok := m["failure"].(string); ok { return f }
	}
	return ""
}
