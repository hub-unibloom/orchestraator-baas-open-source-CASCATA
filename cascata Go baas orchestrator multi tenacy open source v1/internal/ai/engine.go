package ai

import (
	"context"
	"fmt"
	"log/slog"

	"cascata/internal/database"
	"cascata/internal/domain"
)

// Engine is the "Brain" of the Cascata AI BaaS (Phase 28).
// It coordinates context injection and LLM orchestration for tenant-defined intents.
type Engine struct {
	extractor  *ContextExtractor
	projectSvc domain.ProjectResolver
	poolMgr    *database.TenantPoolManager
}

func NewEngine(repo *database.Repository, projectSvc domain.ProjectResolver, poolMgr *database.TenantPoolManager) *Engine {
	return &Engine{
		extractor:  NewContextExtractor(repo),
		projectSvc: projectSvc,
		poolMgr:    poolMgr,
	}
}

// PromptConfig defines the parameters for an AI interaction with injected context.
type PromptConfig struct {
	Intent      string `json:"intent"`
	Temperature float32 `json:"temperature"`
	Model       string `json:"model"`
	UserPrompt  string `json:"user_prompt"`
}

// ExecuteWithContext enriches the user's prompt with real-time tenant data and sends it to the LLM.
func (e *Engine) ExecuteWithContext(ctx context.Context, slug string, config PromptConfig) (string, error) {
	slog.Info("ai: executing intent with context injection", "slug", slug, "intent", config.Intent)

	// 1. Resolve Project Metadata
	p, err := e.projectSvc.Resolve(ctx, slug)
	if err != nil { return "", err }

	// 2. Extract Database Schema Context
	schema, err := e.extractor.GetSchemaContext(ctx, slug)
	if err != nil {
		slog.Warn("ai: schema extraction failed, proceeding with limited context", "error", err)
	}

	// 3. Build Final Multi-Modal Prompt (System + Context + User)
	systemPrompt := fmt.Sprintf(`
		You are the AI Resident of the project '%s'.
		%s
		---
		Tenant Database Schema:
		%s
		---
		Rules:
		- Only provide information based on the provided schema.
		- If the user asks for a query, output valid SQL for PostgreSQL.
		- Do not reveal internal IDs unless specifically asked.
	`, p.Name, p.Metadata["ai_instructions"], schema)

	// Phase 28: Bridge to AI Provider (OpenAI/Anthropic/Gemini)
	// Integration with 'postman' style Dispatcher for AI.
	
	slog.Debug("ai: sending payload to LLM provider", "model", config.Model)
	
	// Simulated response for architectural Phase 28 demonstration
	// In Phase 29/30, this will be the REAL API call using the Tenant's Vault keys.
	return fmt.Sprintf("AI response for %s with injected schema context", slug), nil
}
