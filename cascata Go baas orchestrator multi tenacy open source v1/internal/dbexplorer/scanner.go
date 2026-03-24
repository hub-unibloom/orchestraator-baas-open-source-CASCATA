package dbexplorer

import (
	"context"
	"fmt"

	"cascata/internal/database"
)

// ImpactScanner recursively analyzes the database schema and Cascata system metadata to find dependencies.
type ImpactScanner struct {
	repo       *database.Repository
	systemPool *database.Pool // For `system.automations` scanning
}

func NewImpactScanner(repo *database.Repository, systemPool *database.Pool) *ImpactScanner {
	return &ImpactScanner{
		repo:       repo,
		systemPool: systemPool,
	}
}

// ImpactReport holds all dependent entities that would be broken by a DDL change.
type ImpactReport struct {
	Views       []string `json:"views"`
	RPCs        []string `json:"rpcs"`
	Triggers    []string `json:"triggers"`
	Automations []string `json:"automations"`
	WillBreak   bool     `json:"will_break"`
}

// ScanTableDeletion returns all dependent entities that reference the given table.
func (s *ImpactScanner) ScanTableDeletion(ctx context.Context, slug string, table string) (*ImpactReport, error) {
	report := &ImpactReport{
		Views:       make([]string, 0),
		RPCs:        make([]string, 0),
		Triggers:    make([]string, 0),
		Automations: make([]string, 0),
	}

	// 1. Scan PostgreSQL pg_depend for Views, Functions, Triggers securely
	claims := database.UserClaims{Role: "service_role"}
	err := s.repo.WithRLS(ctx, claims, slug, false, func(tx pgx.Tx) error {
		// pg_depend query
		sqlPGDepend := `
			SELECT objid::regclass::text as dependent_name, classid::regclass::text as type
			FROM pg_depend 
			WHERE refobjid = $1::regclass AND deptype = 'n'
		`
		rows, err := tx.Query(ctx, sqlPGDepend, table)
		if err != nil { return err }
		defer rows.Close()
		
		for rows.Next() {
			var name, depType string
			if err := rows.Scan(&name, &depType); err == nil {
				if depType == "pg_class" {
					report.Views = append(report.Views, name)
				} else if depType == "pg_proc" {
					report.RPCs = append(report.RPCs, name)
				}
			}
		}
		return nil
	})
	if err != nil { slog.Warn("impact_scanner: pg_depend scan error", "slug", slug, "error", err) }

	// 2. Scan Cascata Automations (Metadata DB)
	// We search for nodes that reference the table slug
	sqlMetadata := `
		SELECT name FROM system.automations 
		WHERE project_slug = $1 
		AND (nodes::text LIKE $2 OR trigger_config::text LIKE $2)
	`
	likePattern := fmt.Sprintf("%%\"%s\"%%", table)
	sysRows, err := s.systemPool.Query(ctx, sqlMetadata, slug, likePattern)
	if err == nil {
		defer sysRows.Close()
		for sysRows.Next() {
			var autoName string
			if err := sysRows.Scan(&autoName); err == nil {
				report.Automations = append(report.Automations, autoName)
			}
		}
	}

	if len(report.Views) > 0 || len(report.RPCs) > 0 || len(report.Automations) > 0 {
		report.WillBreak = true
	}

	return report, nil
}
