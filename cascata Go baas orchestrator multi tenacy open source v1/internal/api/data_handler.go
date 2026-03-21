package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"cascata/internal/database"
	"github.com/go-chi/chi/v5"
)

// DataHandler processes dynamic CRUD operations generated from PostgREST syntax API.
type DataHandler struct {
	repo   *database.Repository
	format *ProtocolFormat
}

func NewDataHandler(repo *database.Repository) *DataHandler {
	return &DataHandler{
		repo:   repo,
		format: &ProtocolFormat{},
	}
}

// ServeGet table records
// GET /api/data/{project}/{table}?select=id,name&filter=eq.active
func (h *DataHandler) ServeGet(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "project")
	table := chi.URLParam(r, "table")

	// 1. Parse PostgREST syntax
	pq, err := ParsePostgrest(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// 2. Build Query
	query := fmt.Sprintf("SELECT %s FROM %q", pq.Select, table)
	if pq.Where != "" {
		query += fmt.Sprintf(" WHERE %s", pq.Where)
	}
	if pq.Order != "" {
		query += fmt.Sprintf(" ORDER BY %s", pq.Order)
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", pq.Limit, pq.Offset)

	// 3. Acquire Connection and Execute under RLS
	// For production, this requires setting the specific API role and jwt claims.
	
	// mock database access
	// result, err := h.repo.ExecuteRLS(r.Context(), slug, query, pq.Params)
	result := []map[string]interface{}{}

	// 4. Negotiate Format
	accept := r.Header.Get("Accept")
	if accept == "" {
		accept = "application/json"
	}

	payload, finalType, err := h.format.Encode(result, accept)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", finalType)
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}

// HandleOpenAPI returns the localized OpenAPI schema document.
func (h *DataHandler) HandleOpenAPI(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "project")

	version := r.URL.Query().Get("openapi")
	if version == "" {
		// Try via Accept Header fallback
		accept := r.Header.Get("Accept")
		if accept == "application/vnd.oai.openapi+json;version=3.0" {
			version = "3.0"
		} else if accept == "application/vnd.swagger+json;version=1.0" {
			version = "1.0"
		} else {
			version = "2.0" // Default Swagger for Supabase compatibility
		}
	}

	gen := NewOpenAPIGenerator(h.repo)
	doc, err := gen.Generate(r.Context(), slug, version)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}
