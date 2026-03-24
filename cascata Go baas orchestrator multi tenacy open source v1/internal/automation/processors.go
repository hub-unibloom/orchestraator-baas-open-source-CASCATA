package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cascata/internal/database"
	"cascata/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ProcessNode decodes the configuration and executes the specific node type logic for Phase 10.
func (e *WorkflowEngine) ProcessNode(ctx context.Context, execCtx *ExecutionContext, node *domain.WorkflowNode) (interface{}, error) {
	if node.Config == nil {
		return nil, nil
	}

	switch node.Type {
	case domain.NodeHTTP:
		return e.executeHttp(ctx, execCtx, node)

	case domain.NodeDB:
		return e.executeSql(ctx, execCtx, node)

	case domain.NodeScript:
		// Edge Logic (Phantom JS - TODO: Phase 23 integration with a JS runtime like otto)
		source, _ := node.Config["source"].(string)
		slog.Debug("automation: edge script script", "source", source)
		return map[string]string{"status": "script executed (simulated for Phase 10)"}, nil

	default:
		return nil, fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// executeSql runs a tenant-specific SQL operation with RLS context (Phase 10 Enterprise).
func (e *WorkflowEngine) executeSql(ctx context.Context, execCtx *ExecutionContext, node *domain.WorkflowNode) (interface{}, error) {
	sql, _ := node.Config["sql"].(string)
	if sql == "" { return nil, fmt.Errorf("missing SQL") }

	// Sanity Check for destructive DDL
	forbidden := []string{"DROP ", "TRUNCATE ", "ALTER SYSTEM"}
	upper := strings.ToUpper(sql)
	for _, f := range forbidden {
		if strings.Contains(upper, f) {
			return nil, fmt.Errorf("security: Forbidden DDL in automation: %s", f)
		}
	}

	// EXECUTION: We acquire the pool for the project
	pool, err := execCtx.PoolManager.GetPool(ctx, execCtx.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("automation: failed to resolve pool: %w", err)
	}

	// Automations run as 'service_role' (Admin) but within the Project's Physical Database.
	claims := database.UserClaims{
		Sub:   "automation_engine",
		Email: "automation@cascata.system",
		Role:  "service_role",
	}

	var results []map[string]interface{}
	err = pool.WithRLS(ctx, claims, execCtx.ProjectSlug, false, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, sql)
		if err != nil { return err }
		defer rows.Close()

		for rows.Next() {
			val, _ := rows.Values()
			row := make(map[string]interface{})
			for i, fd := range rows.FieldDescriptions() {
				row[string(fd.Name)] = val[i]
			}
			results = append(results, row)
		}
		return nil
	})

	return results, err
}

// executeHttp performs an outbound request with JSON payload resolution.
func (e *WorkflowEngine) executeHttp(ctx context.Context, execCtx *ExecutionContext, node *domain.WorkflowNode) (interface{}, error) {
	targetUrl, _ := node.Config["url"].(string)
	targetUrl = ResolveVariables(targetUrl, execCtx.Vars)

	if targetUrl == "" { return nil, fmt.Errorf("missing target URL") }

	method, _ := node.Config["method"].(string)
	if method == "" { method = "POST" }

	body, _ := json.Marshal(e.ResolveObject(node.Config["body"], execCtx.Vars))
	
	req, err := http.NewRequestWithContext(ctx, method, targetUrl, bytes.NewBuffer(body))
	if err != nil { return nil, err }
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cascata-Trigger", "workflow")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upstream responded with HTTP %d", resp.StatusCode)
	}

	var resData interface{}
	b, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(b, &resData)

	return resData, nil
}
