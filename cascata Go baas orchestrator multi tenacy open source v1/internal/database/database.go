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

// Repository manages database interactions for metadata.
type Repository struct {
	Pool *pgxpool.Pool
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
			atomicSQL := fmt.Sprintf(`
				SET LOCAL ROLE %s;
				SET LOCAL statement_timeout = $1;
				SET LOCAL "cascata.project_slug" = $2;
				SET LOCAL "cascata.isolation_scope" = 'tenant';
				SET LOCAL "request.jwt.claim.sub" = $3;
				SET LOCAL "request.jwt.claim.email" = $4;
				SET LOCAL "request.jwt.claim.role" = $5;
			`, role)

			if _, err := tx.Exec(ctx, atomicSQL, timeout, projectSlug, claims.Sub, claims.Email, claims.Role); err != nil {
				slog.Error("security context injection failed", "slug", projectSlug, "err", err)
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

// NewTenantPool creates a new connection pool for a specific physical database on the current host.
func NewTenantPool(ctx context.Context, masterURL, dbName string) (*Repository, error) {
	// 1. Identify current host and port from master URL.
	// pgx.ParseConfig helps extract components.
	cfg, err := pgxpool.ParseConfig(masterURL)
	if err != nil {
		return nil, fmt.Errorf("database.NewTenantPool: parse: %w", err)
	}

	// 2. Swapping Database name to the tenant-specific one.
	cfg.ConnConfig.Database = dbName

	// 3. Tuning the tenant pool: more restrictive than the metadata pool.
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 15 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("database.NewTenantPool: create pool [%s]: %w", dbName, err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database.NewTenantPool: ping [%s]: %w", dbName, err)
	}

	slog.Info("tenant database pool established", "dbName", dbName)
	return &Repository{Pool: pool}, nil
}
