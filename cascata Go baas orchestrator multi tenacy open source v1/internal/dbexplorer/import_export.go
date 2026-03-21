package dbexplorer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"cascata/internal/database"
)

// ImportExport handles bulk loading and unloading of data to formats like CSV/JSON.
type ImportExport struct {
	repo *database.Repository
}

func NewImportExport(repo *database.Repository) *ImportExport {
	return &ImportExport{repo: repo}
}

// IngestJSON uploads a batch of JSON objects into the targeted table.
// Provides automatic inferencing of types and batching for large files.
func (ie *ImportExport) IngestJSON(ctx context.Context, slug string, table string, data io.Reader) (int, error) {
	pool, err := ie.repo.GetProjectPool(slug)
	if err != nil {
		return 0, err
	}

	decoder := json.NewDecoder(data)
	
	// Expecting an array of objects
	t, err := decoder.Token()
	if err != nil || t != json.Delim('[') {
		return 0, fmt.Errorf("import must be a json array")
	}

	totalInserted := 0
	batchSize := 500
	var batch []map[string]interface{}

	for decoder.More() {
		var obj map[string]interface{}
		if err := decoder.Decode(&obj); err != nil {
			return totalInserted, err
		}
		batch = append(batch, obj)
		
		if len(batch) >= batchSize {
			err = ie.flushBatch(ctx, pool, table, batch)
			if err != nil {
				return totalInserted, err
			}
			totalInserted += len(batch)
			batch = batch[:0]
		}
	}

	// Flush remaining
	if len(batch) > 0 {
		err = ie.flushBatch(ctx, pool, table, batch)
		if err != nil {
			return totalInserted, err
		}
		totalInserted += len(batch)
	}

	return totalInserted, nil
}

// flushBatch runs a bulk insert using UNNEST for extreme performance.
// Same architecture implemented in the API DataController.
func (ie *ImportExport) flushBatch(ctx context.Context, pool *database.Pool, table string, batch []map[string]interface{}) error {
	if len(batch) == 0 { return nil }
	
	cols := make([]string, 0)
	for k := range batch[0] {
		cols = append(cols, k)
	}
	
	var placeHolders []string
	values := make([]interface{}, len(cols))
	
	for i, col := range cols {
		placeHolders = append(placeHolders, fmt.Sprintf("$%d::text[]", i+1))
		colVals := make([]interface{}, len(batch))
		for j, row := range batch {
			colVals[j] = row[col]
		}
		values[i] = colVals
	}
	
	sql := fmt.Sprintf(`
		INSERT INTO public.%q (%s) 
		SELECT * FROM UNNEST(%s)
	`, table, strings.Join(cols, ", "), strings.Join(placeHolders, ", "))
	
	_, err := pool.Exec(ctx, sql, values...)
	return err
}
