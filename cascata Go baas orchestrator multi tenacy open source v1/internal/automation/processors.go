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

	"cascata/internal/ai"
	"cascata/internal/database"
	"cascata/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ProcessNode decodes the configuration and executes the specific node type logic for Phase 10.
func (e *WorkflowEngine) ProcessNode(ctx context.Context, execCtx *ExecutionContext, node *domain.WorkflowNode) (interface{}, error) {
	if node.Config == nil {
		return nil, nil
	}

	// Resolve variables in the node's config properties (Foundation Phase 10)
	props := e.ResolveObject(node.Config, execCtx.Vars).(map[string]interface{})

	switch node.Type {
	case domain.NodeHTTP:
		return e.executeHttp(ctx, execCtx, node)

	case domain.NodeDB:
		return e.executeSql(ctx, execCtx, node)

	case "EMAIL":
		// Legacy mapping for EMAIL nodes
		to := fmt.Sprintf("%v", props["to"])
		subject := fmt.Sprintf("%v", props["subject"])
		body := fmt.Sprintf("%v", props["body"])
		err := e.postman.SendEmail(ctx, to, subject, body)
		return map[string]string{"status": "sent"}, err

	case "WHATSAPP":
		to := fmt.Sprintf("%v", props["to"])
		msg := fmt.Sprintf("%v", props["message"])
		err := e.postman.SendWhatsApp(ctx, to, msg)
		return map[string]string{"status": "dispatched"}, err

	case "DB_QUERY":
		sql := fmt.Sprintf("%v", props["sql"])
		pool, err := execCtx.PoolManager.GetPool(ctx, execCtx.ProjectSlug, execCtx.DBName, execCtx.MaxConns)
		if err != nil { return nil, err }
		
		rows, err := pool.Pool.Query(ctx, sql)
		if err != nil { return nil, err }
		defer rows.Close()
		
		return map[string]string{"status": "executed (synced pool)"}, nil

	case "PUSH_NOTIFICATION":
		token := fmt.Sprintf("%v", props["token"])
		title := fmt.Sprintf("%v", props["title"])
		body := fmt.Sprintf("%v", props["body"])
		
		var data map[string]string
		if d, ok := props["data"].(map[string]interface{}); ok {
			data = make(map[string]string)
			for k, v := range d { data[k] = fmt.Sprintf("%v", v)}
		}

		err := e.postman.SendPush(ctx, token, title, body, data)
		return map[string]string{"status": "dispatched_to_hub"}, err

	case "WEBHOOK":
		// Simplified Webhook shortcut
		return e.executeHttp(ctx, execCtx, node)

	case "ASK_AI":
		prompt := fmt.Sprintf("%v", props["prompt"])
		config := ai.PromptConfig{
			Intent:     fmt.Sprintf("%v", props["intent"]),
			Model:      fmt.Sprintf("%v", props["model"]),
			UserPrompt: prompt,
		}
		if config.Model == "" { config.Model = "gpt-4o" }

		response, err := e.aiEngine.ExecuteWithContext(ctx, execCtx.ProjectSlug, config)
		if err != nil {
			return nil, fmt.Errorf("automation.ai: %w", err)
		}
		return map[string]string{"response": response}, nil

	case "REALTIME_BROADCAST":
		payload := props["payload"]
		if payload == nil { payload = execCtx.Vars["$last"] }
		err := e.realtime.Broadcast(ctx, execCtx.ProjectSlug, payload)
		return map[string]string{"status": "broadcasted"}, err

	case domain.NodeScript:
		// Edge Logic (Phantom JS - Phase 23 integration)
		source, _ := props["source"].(string)
		slog.Debug("automation: edge script script", "source", source)
		return map[string]string{"status": "script executed (simulated)"}, nil

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
	pool, err := execCtx.PoolManager.GetPool(ctx, execCtx.ProjectSlug, execCtx.DBName, execCtx.MaxConns)
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
