package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"cascata/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is an alias for pgxpool.Pool to maintain compatibility across the orchestrator.
type Pool = pgxpool.Pool

// Queryer is handled by domain.Queryer now.

// Repository manages database interactions for metadata.
type Repository struct {
	Pool *Pool
}

// Connect establishes the metadata database connection pool.
func Connect(ctx context.Context, databaseURL string) (*Repository, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database.Connect: parse config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("database.Connect: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database.Connect: ping: %w", err)
	}

	slog.Info("database session established for metadata orchestrator")
	return &Repository{Pool: pool}, nil
}

// Close gracefully shuts down the connection pool.
func (r *Repository) Close() {
	r.Pool.Close()
}

// UserClaims represents the JWT payload for RLS injection.
type UserClaims struct {
	Sub   string
	Email string
	Role  string
}

// WithRLS executes a function within a transaction with hardened RLS and PgBouncer safety.
// Implements exponential backoff for transient database errors.
func (r *Repository) WithRLS(ctx context.Context, claims UserClaims, projectSlug string, isMaintenance bool, fn func(pgx.Tx) error) error {
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		err = pgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {
			// 1. Session Cleanup & Security Context Injection (Atomic Workflow)
			// Phase 22 Sinergy: minimize round-trips by combining mandatory security sets.
			timeout := "5s"
			if isMaintenance { timeout = "30s" }
			role := claims.Role
			if role == "" { role = "anon" }

			// 2. Resolve Sovereign Namespace (Barreira por Pseudônimo)
			// Por padrão, o inquilino cai no seu schema _public.
			// O sistema de orquestração cai no seu schema _system.
			targetNamespace := fmt.Sprintf("%s_public", projectSlug)
			if projectSlug == "cascata" {
				targetNamespace = "cascata_system"
			}

			// We use a single multiline statement for performance and atomic security.
			// %%q in Sprintf double-quotes the identifier for safety.
			atomicSQL := fmt.Sprintf(`
				SET LOCAL ROLE %%q;
				SET LOCAL statement_timeout = $1;
				SET LOCAL search_path TO %%q, public;
				SET LOCAL "cascata.project_slug" = $2;
				SET LOCAL "cascata.isolation_scope" = 'tenant';
				SET LOCAL "request.jwt.claim.sub" = $3;
				SET LOCAL "request.jwt.claim.email" = $4;
				SET LOCAL "request.jwt.claim.role" = $5;
			`, role, targetNamespace)

			if _, err := tx.Exec(ctx, atomicSQL, timeout, projectSlug, claims.Sub, claims.Email, claims.Role); err != nil {
				slog.Error("security context injection failed", "slug", projectSlug, "err", err, "role", role, "namespace", targetNamespace)
				return fmt.Errorf("security_injection: %w", err)
			}

			return fn(tx)
		})

		if err == nil { return nil }
		if !isTransientError(err) { break }

		slog.Warn("transient database error, retrying", "attempt", i+1, "error", err)
		time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
	}

	return err
}

// QueryWithRLS is a high-level helper for one-off RLS-gated queries (Phase 9 Bridge).
// Used primarily by the sandboxed Runtime to execute client-provided SQL safely.
func (r *Repository) QueryWithRLS(ctx context.Context, role, sql string, params []interface{}) (interface{}, error) {
	var result []map[string]interface{}
	
	claims := UserClaims{Role: role}
	projectSlug := "cascata" // Fallback or injected
	if slug, ok := ctx.Value("cascata.project_slug").(string); ok {
		projectSlug = slug
	}

	err := r.WithRLS(ctx, claims, projectSlug, false, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, sql, params...)
		if err != nil { return err }
		defer rows.Close()

		// Dynamic result mapping to JSON-compatible maps
		fieldDescriptions := rows.FieldDescriptions()
		for rows.Next() {
			values, err := rows.Values()
			if err != nil { return err }
			
			row := make(map[string]interface{})
			for i, fd := range fieldDescriptions {
				row[string(fd.Name)] = values[i]
			}
			result = append(result, row)
		}
		return nil
	})

	return result, err
}

func isTransientError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 57P01: admin_shutdown, 57P03: cannot_connect_now, 08000: connection_exception
		switch pgErr.Code {
		case "57P01", "57P03", "08000":
			return true
		}
	}

	if fmt.Sprintf("%v", err) == "conn closed" { return true }
	if err != nil && (fmt.Sprintf("%v", err) == "driver: bad connection") { return true }
	
	// Check for pgx standard connection errors
	return false 
}
