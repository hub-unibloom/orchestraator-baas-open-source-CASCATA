package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// SyntheticGenerator leverages the AI Engine to generate high-fidelity test data.
// It acts as the "Genesis Lab" (Phase 32).
type SyntheticGenerator struct {
	engine *Engine
}

func NewSyntheticGenerator(engine *Engine) *SyntheticGenerator {
	return &SyntheticGenerator{engine: engine}
}

// GenerateRecords creates multiple JSON objects matching a table's schema.
func (g *SyntheticGenerator) GenerateRecords(ctx context.Context, slug, table string, count int) ([]map[string]interface{}, error) {
	slog.Info("ai.synthetic: generating test data", "table", table, "count", count)

	prompt := fmt.Sprintf("Generate %d realistic JSON records for the table '%s'. Use the provided schema context.", count, table)
	
	config := PromptConfig{
		Intent:     "DATA_SEEDING",
		Model:      "gpt-4o",
		UserPrompt: prompt,
	}

	raw, err := g.engine.ExecuteWithContext(ctx, slug, config)
	if err != nil {
		return nil, err
	}

	slog.Debug("ai.synthetic: records generated via LLM", "response_len", len(raw))

	// In Phase 32, we assume LLM returns valid JSON array.
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		// Log error but don't fail, maybe the engine returned raw text.
		// For Cascata v1.0.0.0 we enforce strict JSON structure.
		slog.Warn("ai.synthetic: LLM response did not contain valid JSON array", "error", err)
		return nil, fmt.Errorf("ai: engine returned invalid data structure: %w", err)
	}

	return records, nil
}
