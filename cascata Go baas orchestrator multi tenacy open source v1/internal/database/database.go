package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is an alias for pgxpool.Pool to maintain compatibility across the orchestrator.
type Pool = pgxpool.Pool

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

			// We use a single multiline statement for performance and atomic security.
			// %%q in Sprintf double-quotes the identifier for safety.
			atomicSQL := fmt.Sprintf(`
				SET LOCAL ROLE %%q;
				SET LOCAL statement_timeout = $1;
				SET LOCAL search_path TO "%%s", public;
				SET LOCAL "cascata.project_slug" = $2;
				SET LOCAL "cascata.isolation_scope" = 'tenant';
				SET LOCAL "request.jwt.claim.sub" = $3;
				SET LOCAL "request.jwt.claim.email" = $4;
				SET LOCAL "request.jwt.claim.role" = $5;
			`, role, projectSlug)

			if _, err := tx.Exec(ctx, atomicSQL, timeout, projectSlug, claims.Sub, claims.Email, claims.Role); err != nil {
				slog.Error("security context injection failed", "slug", projectSlug, "err", err, "role", role)
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

// GetTables retrieves the list of base tables in the specified schema.
func (r *Repository) GetTables(ctx context.Context, schema string) ([]string, error) {
	if schema == "" {
		schema = "public"
	}

	rows, err := r.Pool.Query(ctx, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = $1 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`, schema)
	if err != nil {
		return nil, fmt.Errorf("database.GetTables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

// ColumnMetadata defines the structural pulse of a database column.
type ColumnMetadata struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	IsNullable   bool        `json:"is_nullable"`
	DefaultValue interface{} `json:"default_value"`
	IsPrimaryKey bool        `json:"is_primary_key"`
}

// GetColumns retrieves the structural definition of a table's columns.
func (r *Repository) GetColumns(ctx context.Context, schema, table string) ([]ColumnMetadata, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT 
			c.column_name, 
			c.udt_name, 
			c.is_nullable = 'YES',
			c.column_default,
			EXISTS (
				SELECT 1 FROM information_schema.key_column_usage kcu
				JOIN information_schema.table_constraints tc ON kcu.constraint_name = tc.constraint_name
				WHERE kcu.table_name = c.table_name 
				AND kcu.column_name = c.column_name 
				AND tc.constraint_type = 'PRIMARY KEY'
			) as is_pk
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := r.Pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("database.GetColumns: %w", err)
	}
	defer rows.Close()

	var columns []ColumnMetadata
	for rows.Next() {
		var col ColumnMetadata
		if err := rows.Scan(&col.Name, &col.Type, &col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey); err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}
// FetchRows retrieves data from a specific table with dynamic row scanning into generic maps.
func (r *Repository) FetchRows(ctx context.Context, table string, limit int) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %q LIMIT %d", table, limit)

	rows, err := r.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("database.FetchRows: %w", err)
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	var results []map[string]interface{}

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		entry := make(map[string]interface{})
		for i, fd := range fieldDescriptions {
			entry[string(fd.Name)] = values[i]
		}
		results = append(results, entry)
	}

	return results, nil
}
