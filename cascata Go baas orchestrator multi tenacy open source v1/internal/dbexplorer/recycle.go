package dbexplorer

import (
	"context"
	"fmt"
	"time"

	"cascata/internal/database"
)

// RecycleBin manages safedeletion and restoration of tables.
type RecycleBin struct {
	repo *database.Repository
}

func NewRecycleBin(repo *database.Repository) *RecycleBin {
	return &RecycleBin{repo: repo}
}

// SoftDelete renames a table to _deleted_{TIMESTAMP}_tablename so it can be restored later.
func (r *RecycleBin) SoftDelete(ctx context.Context, slug string, table string) error {
	pool, err := r.repo.GetProjectPool(slug)
	if err != nil {
		return err
	}

	deletedName := fmt.Sprintf("_deleted_%d_%s", time.Now().UnixMilli(), table)
	
	sql := fmt.Sprintf("ALTER TABLE public.%q RENAME TO %q", table, deletedName)
	
	_, err = pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("recycle bin delete failed: %w", err)
	}
	
	return nil
}

// Restore recovers a previously soft-deleted table to its original name.
func (r *RecycleBin) Restore(ctx context.Context, slug string, deletedName string) (string, error) {
	pool, err := r.repo.GetProjectPool(slug)
	if err != nil {
		return "", err
	}

	// Extract original name by stripping prefix
	var originalName string
	// format is _deleted_1234567890_tablename
	_, err = fmt.Sscanf(deletedName, "_deleted_%d_%s", new(int64), &originalName)
	if err != nil {
		// fallback naive split
		parts := []rune(deletedName)
		// find the second underscore
		underscoreCount := 0
		cutIdx := 0
		for i, c := range parts {
			if c == '_' {
				underscoreCount++
				if underscoreCount == 3 {
					cutIdx = i + 1
					break
				}
			}
		}
		originalName = string(parts[cutIdx:])
	}

	sql := fmt.Sprintf("ALTER TABLE public.%q RENAME TO %q", deletedName, originalName)
	
	_, err = pool.Exec(ctx, sql)
	if err != nil {
		return "", fmt.Errorf("recycle bin restore failed: %w", err)
	}

	return originalName, nil
}
