package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
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
func (r *Repository) WithRLS(ctx context.Context, claims UserClaims, projectSlug string, isMaintenance bool, fn func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {
		// 1. Session Cleanup (PgBouncer safety and Zero-Trust start)
		if _, err := tx.Exec(ctx, "RESET ALL;"); err != nil {
			return fmt.Errorf("database.WithRLS: reset session: %w", err)
		}

		// 2. Dynamic Statement Timeout.
		timeout := "5s"         // Aggressive for API (Tenant users)
		if isMaintenance {
			timeout = "30s"      // Relaxed for maintenance
		}

		// 3. Multi-Key Security Injection (Transaction Pipeline logic).
		// We concatenate all SET LOCAL into a single EXEC to minimize RTT.
		setSql := `
			SET LOCAL statement_timeout = $1;
			SET LOCAL ROLE %s;
			SET LOCAL "cascata.project_slug" = $2;
			SET LOCAL "request.jwt.claim.sub" = $3;
			SET LOCAL "request.jwt.claim.email" = $4;
			SET LOCAL "request.jwt.claim.role" = $5;
		`
		
		// Secure parameter interpolation for the session variables.
		finalSql := fmt.Sprintf(setSql, claims.Role)
		_, err := tx.Exec(ctx, finalSql, timeout, projectSlug, claims.Sub, claims.Email, claims.Role)
		
		if err != nil {
			// Fail-Closed: Log failure and abort immediately. 
			// pgx.BeginFunc handles the Rollback automatically.
			slog.Error("CRITICAL: security context injection failed", "slug", projectSlug, "error", err)
			return fmt.Errorf("database.WithRLS: fail-closed: injection failed: %w", err)
		}

		return fn(tx)
	})
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
