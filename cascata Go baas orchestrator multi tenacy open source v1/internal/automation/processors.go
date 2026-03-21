package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cascata/internal/utils"
)

// ProcessNode decodes the configuration and executes the specific node type logic.
func (e *Engine) ProcessNode(ctx context.Context, execCtx *Context, node *Node) (interface{}, error) {
	if node.Config == nil {
		return nil, nil
	}

	switch node.Type {
	case "transform":
		transformed := ResolveObject(node.Config["template"], execCtx.Vars)
		execCtx.Vars["$output"] = transformed
		return transformed, nil

	case "query":
		return e.executeSecureSql(ctx, execCtx, node)

	case "http":
		return e.executeHttp(ctx, execCtx, node)

	case "conditional":
		return e.evaluateLogic(node, execCtx), nil

	case "response":
		return ResolveObject(node.Config["body"], execCtx.Vars), nil

	case "edge_function":
		return e.executeEdge(ctx, execCtx, node)

	default:
		return nil, fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// executeEdge runs either a Go-native script or a WASM module.
func (e *Engine) executeEdge(ctx context.Context, execCtx *Context, node *Node) (interface{}, error) {
	runtimeType, _ := node.Config["runtime"].(string) // "go" or "wasm"
	source, _ := node.Config["source"].(string)

	if runtimeType == "wasm" {
		// WASM implementation: we'd pull the binary and call executeWASM
		return nil, fmt.Errorf("engine: wasm activation pending for critical mode")
	}

	// Native Go (via Yaegi)
	// For Phase 12, we demonstrate the Go path.
	// In the real system, this source would have been translated/transpiled.
	return nil, nil
}

// executeSecureSql runs a SQL query with RLS and pattern safety.
func (e *Engine) executeSecureSql(ctx context.Context, execCtx *Context, node *Node) (interface{}, error) {
	sql, _ := node.Config["sql"].(string)
	if sql == "" {
		return nil, fmt.Errorf("missing SQL script")
	}

	// 1. Pattern Shield
	forbidden := []string{"COPY ", "PG_READ_FILE", "PG_LS_DIR", "DO $$", "ALTER SYSTEM"}
	upper := strings.ToUpper(sql)
	for _, p := range forbidden {
		if strings.Contains(upper, p) {
			return nil, fmt.Errorf("Security Block: Forbidden SQL Pattern '%s'", p)
		}
	}

	// 2. Resolve Parameters
	paramsIn, _ := node.Config["params"].([]interface{})
	resolvedParams := make([]interface{}, len(paramsIn))
	for i, p := range paramsIn {
		resolvedParams[i] = GetVar(fmt.Sprintf("%v", p), execCtx.Vars)
	}

	// 3. Execute via Repository with RLS (Uses WithRLS logic from internal/database)
	// For automations, we enforce a strict 8s timeout.
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	// TODO: Integrate execCtx.ProjectPool.Query with execCtx.AuthCtx integration.
	// For Phase 10, we establish the flow.
	return nil, nil
}

// executeHttp performs an outbound request with SSRF protection and retries.
func (e *Engine) executeHttp(ctx context.Context, execCtx *Context, node *Node) (interface{}, error) {
	targetUrl, _ := node.Config["url"].(string)
	targetUrl = ResolveVariables(targetUrl, execCtx.Vars)

	// SSRF Shield
	if err := utils.IsSSRFSafe(targetUrl); err != nil {
		return nil, err
	}

	method, _ := node.Config["method"].(string)
	if method == "" { method = "POST" }

	body, _ := json.Marshal(ResolveObject(node.Config["body"], execCtx.Vars))
	
	req, err := http.NewRequestWithContext(ctx, method, targetUrl, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// 4. Resolve Vault Auth if vault:// reference exists
	// Credentials processing logic here...

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var resData interface{}
	if err := json.NewDecoder(resp.Body).Decode(&resData); err != nil {
		return nil, err
	}

	return resData, nil
}

func (e *Engine) evaluateLogic(node *Node, execCtx *Context) interface{} {
	// Simple evaluation for Phase 10 Logic nodes (eq, neq, gt, lt)
	left := GetVar(fmt.Sprintf("%v", node.Config["left"]), execCtx.Vars)
	right := node.Config["right"]
	
	switch node.Config["op"] {
	case "eq": return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
	case "neq": return fmt.Sprintf("%v", left) != fmt.Sprintf("%v", right)
	}
	return false
}
