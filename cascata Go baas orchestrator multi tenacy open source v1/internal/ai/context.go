package ai

import (
	"context"
	"fmt"
	"strings"

	"cascata/internal/database"
)

// ContextExtractor pulls schema information to feed the LLM prompt.
type ContextExtractor struct {
	repo *database.Repository
}

func NewContextExtractor(repo *database.Repository) *ContextExtractor {
	return &ContextExtractor{repo: repo}
}

// GetSchemaContext returns a textual representation of all tables and columns.
func (e *ContextExtractor) GetSchemaContext(ctx context.Context, slug string) (string, error) {
	// We need to query the project's database (public schema)
	// For Phase 15, we pull from information_schema.
	
	sql := `
		SELECT table_name, column_name, data_type 
		FROM information_schema.columns 
		WHERE table_schema = 'public'
		ORDER BY table_name, ordinal_position
	`
	// Assuming l.repo has a way to get a pool for a slug (already implemented in Phase 2)
	pool := e.repo.Pool // In some cases, we might want the specific tenant pool
	
	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return "", fmt.Errorf("ai: context extraction failed: %w", err)
	}
	defer rows.Close()

	var builder strings.Builder
	builder.WriteString("Current Database Schema:\n")
	
	currentTable := ""
	for rows.Next() {
		var table, col, dtype string
		if err := rows.Scan(&table, &col, &dtype); err != nil {
			return "", err
		}
		
		if table != currentTable {
			builder.WriteString(fmt.Sprintf("\n- Table: %s\n", table))
			currentTable = table
		}
		builder.WriteString(fmt.Sprintf("  * %s (%s)\n", col, dtype))
	}

	return builder.String(), nil
}
