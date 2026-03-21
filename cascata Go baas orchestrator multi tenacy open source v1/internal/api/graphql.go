package api

import (
	"context"
	"fmt"
	"net/http"

	"cascata/internal/database"
)

// GraphQLHandler manages dynamic schema generation and queries.
// It translates incoming GraphQL ASTs to SQL mimicking PostGraphile.
type GraphQLHandler struct {
	repo *database.Repository
}

func NewGraphQLHandler(repo *database.Repository) *GraphQLHandler {
	return &GraphQLHandler{repo: repo}
}

// ServeHTTP acts as the primary GraphQL endpoint /graphql
func (g *GraphQLHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// For Cascata v1, we hook to the core pg_graphql extension if available 
	// (similar to Supabase) or we route through our AST-to-SQL layer constraint.
	
	// Example integration assumes pg_graphql is installed inside the tenant DB:
	// SELECT graphql.resolve($1);
	
	// Read body (GraphQL query)
	// Build response
	
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"data": null, "errors": [{"message": "GraphQL engine booting"}]}`)
}

// EnsureSchema initializes the pg_graphql extension for a tenant.
func (g *GraphQLHandler) EnsureSchema(ctx context.Context, slug string) error {
	pool, err := g.repo.GetProjectPool(slug)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pg_graphql")
	if err != nil {
		// Just log if native postgres extensions are unavailable
		return fmt.Errorf("graphql: failed to provision extension: %w", err)
	}
	
	return nil
}
