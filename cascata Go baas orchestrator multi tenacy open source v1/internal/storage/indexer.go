package storage

import (
	"context"
	"fmt"
	"path"
	"time"

	"cascata/internal/database"
)

// IndexItem contains the file/folder metadata for auditing.
type IndexItem struct {
	Size     int64
	MimeType string
	IsFolder bool
	Provider string
}

// Indexer manages the persistence of storage objects.
type Indexer struct {
	repo *database.Repository
}

func NewIndexer(repo *database.Repository) *Indexer {
	return &Indexer{repo: repo}
}

// IndexObject inserts or updates an object entry in the system registry.
func (i *Indexer) IndexObject(ctx context.Context, projectSlug, bucket, fullPath string, meta IndexItem) error {
	name := path.Base(fullPath)
	parentPath := path.Dir(fullPath)
	if parentPath == "." {
		parentPath = ""
	}

	sql := `
		INSERT INTO system.storage_objects 
		(project_slug, bucket, name, parent_path, full_path, is_folder, size, mime_type, provider, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (project_slug, bucket, full_path) 
		DO UPDATE SET size = EXCLUDED.size, updated_at = EXCLUDED.updated_at, provider = EXCLUDED.provider
	`
	_, err := i.repo.Pool.Exec(ctx, sql, projectSlug, bucket, name, parentPath, fullPath, meta.IsFolder, meta.Size, meta.MimeType, meta.Provider, time.Now())
	if err != nil {
		return fmt.Errorf("storage.Indexer.Index: %w", err)
	}
	return nil
}

// UnindexObject removes an object or an entire folder tree from the registry.
func (i *Indexer) UnindexObject(ctx context.Context, projectSlug, bucket, fullPath string) error {
	sql := `
		DELETE FROM system.storage_objects 
		WHERE project_slug = $1 AND bucket = $2 AND (full_path = $3 OR full_path LIKE $4)
	`
	_, err := i.repo.Pool.Exec(ctx, sql, projectSlug, bucket, fullPath, fullPath+"/%")
	if err != nil {
		return fmt.Errorf("storage.Indexer.Unindex: %w", err)
	}
	return nil
}

// List returns children of a specific path.
func (i *Indexer) List(ctx context.Context, projectSlug, bucket, parentPath string) ([]map[string]interface{}, error) {
	sql := `
		SELECT name, is_folder, size, updated_at, full_path 
		FROM system.storage_objects 
		WHERE project_slug = $1 AND bucket = $2 AND parent_path = $3
		ORDER BY is_folder DESC, name ASC
	`
	rows, err := i.repo.Pool.Query(ctx, sql, projectSlug, bucket, parentPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var name, fullPath string
		var isFolder bool
		var size int64
		var updatedAt time.Time
		if err := rows.Scan(&name, &isFolder, &size, &updatedAt, &fullPath); err != nil {
			return nil, err
		}
		
		item := map[string]interface{}{
			"name":       name,
			"is_folder":  isFolder,
			"size":       size,
			"updated_at": updatedAt,
			"full_path":  fullPath,
		}
		result = append(result, item)
	}
	return result, nil
}
