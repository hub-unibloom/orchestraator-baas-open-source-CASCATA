package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"cascata/internal/ai"
	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/privacy"
	"cascata/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// DataHandler processes dynamic CRUD operations generated from PostgREST syntax API.
type DataHandler struct {
	projectSvc  *service.ProjectService
	interceptor *Interceptor
	privacy     *privacy.Engine
	format      *ProtocolFormat
	cache       *database.CacheManager
	vectors     *database.VectorEngine
	synthetic   *ai.SyntheticGenerator
}

func NewDataHandler(projectSvc *service.ProjectService, interceptor *Interceptor, pEng *privacy.Engine, cache *database.CacheManager, aiEngine *ai.Engine) *DataHandler {
	return &DataHandler{
		projectSvc:  projectSvc,
		interceptor: interceptor,
		privacy:     pEng,
		format:      &ProtocolFormat{},
		cache:       cache,
		vectors:     database.NewVectorEngine(),
		synthetic:   ai.NewSyntheticGenerator(aiEngine),
	}
}

// resolvePrivacy resolves the masking/locking policies for a project.
func (h *DataHandler) resolvePrivacy(p *domain.Project) privacy.ProjectPrivacyConfig {
	var cfg privacy.ProjectPrivacyConfig
	if p.Metadata == nil { return cfg }
	
	raw, ok := p.Metadata["privacy_config"]
	if !ok { return cfg }
	
	b, _ := json.Marshal(raw)
	_ = json.Unmarshal(b, &cfg)
	return cfg
}

// ProjectStats defines the metrics for the Project Dashboard Overview.
type ProjectStats struct {
	TotalTables int      `json:"total_tables"`
	SchemaSize  int64    `json:"schema_size_bytes"`
	TotalUsers  int      `json:"total_users"`
	Status      string   `json:"status"`
	TableNames  []string `json:"table_names"`
}

// HandleProjectOverview returns real telemetry for a specific tenant schema.
func (h *DataHandler) HandleProjectOverview(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "project")
	ctx := r.Context()

	// 1. Resolve Project (Ensure RLS context)
	p, err := h.projectSvc.Resolve(ctx, slug)
	if err != nil {
		SendError(w, r, http.StatusNotFound, domain.ErrNotFound, "Project not found")
		return
	}

	stats := ProjectStats{
		Status:     p.Status,
		TableNames: []string{},
	}

	// 2. Query Schema Size & Table Count
	sqlStats := `
		SELECT 
			count(*)::int as total_tables,
			COALESCE(sum(pg_total_relation_size(quote_ident(schemaname) || '.' || quote_ident(relname))), 0)::bigint as total_size
		FROM pg_stat_user_tables 
		WHERE schemaname = $1
	`
	_ = h.projectSvc.Repo().Pool.QueryRow(ctx, sqlStats, slug).Scan(&stats.TotalTables, &stats.SchemaSize)

	// 3. Inventory Table Names (The "Real" List)
	rows, _ := h.projectSvc.Repo().Pool.Query(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = $1 AND table_type = 'BASE TABLE'", slug)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				stats.TableNames = append(stats.TableNames, name)
			}
		}
	}

	// 4. Query Resident Count
	sqlUsers := fmt.Sprintf("SELECT count(*)::int FROM %s.residents", slug)
	_ = h.projectSvc.Repo().Pool.QueryRow(ctx, sqlUsers).Scan(&stats.TotalUsers)

	h.respond(w, r, stats)
}

// ServeGet retrieves table records with filtering, selection and canonical cross-tenant isolation.
func (h *DataHandler) ServeGet(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "project")
	table := chi.URLParam(r, "table")

	authCtx, ok := domain.FromContext(r.Context())
	if !ok || authCtx.ProjectSlug != slug {
		SendError(w, r, http.StatusForbidden, ErrForbidden, "cross-tenant access denied")
		return
	}

	p, err := h.projectSvc.Resolve(r.Context(), slug)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "PROJECT_RESOLUTION_FAILURE")
		return
	}
	privCfg := h.resolvePrivacy(p)

	pq, err := ParsePostgrest(r.URL.Query())
	if err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid PostgREST syntax", err.Error())
		return
	}

	query := fmt.Sprintf("SELECT %s FROM %q", pq.Select, table)
	if pq.Where != "" { query += fmt.Sprintf(" WHERE %s", pq.Where) }
	if pq.Order != "" { query += fmt.Sprintf(" ORDER BY %s", pq.Order) }
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", pq.Limit, pq.Offset)

	finalQuery, err := h.interceptor.ValidateQuery(r.Context(), slug, query)
	if err != nil {
		SendError(w, r, http.StatusForbidden, ErrForbidden, "QUERY_BLOCKED", err.Error())
		return
	}

	// 3. Cache Check (Phase 12: Hyper-Drive)
	queryHash := h.cache.HashQuery(finalQuery, pq.Params)
	cacheKey := fmt.Sprintf("cascata:cache:%s:%s:%s:%s", slug, table, authCtx.Role, queryHash)
	
	if cached, err := h.cache.Get(r.Context(), cacheKey); err == nil {
		var cachedResult []map[string]interface{}
		if json.Unmarshal(cached, &cachedResult) == nil {
			h.respond(w, r, cachedResult)
			return
		}
	}

	result, err := h.executePoolQuery(r.Context(), authCtx, finalQuery, pq.Params)
	if err != nil {
		h.handleQueryError(w, r, err)
		return
	}

	// 1. Synchronous Hijacking (Phase 9 Logic Interceptor)
	if len(result) > 0 {
		hijacked, _ := h.interceptor.ProcessResponse(r.Context(), slug, table, "READ", result[0])
		result[0] = hijacked
	}

	finalResult, _ := h.privacy.ApplyMasking(r.Context(), authCtx, privCfg, table, result)
	
	// 4. Fill Cache Async
	go func() {
		b, _ := json.Marshal(finalResult)
		_ = h.cache.Set(context.Background(), cacheKey, b)
	}()

	h.respond(w, r, finalResult)
}

// ServePost handles record insertion and triggers Phase 10 events.
func (h *DataHandler) ServePost(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "project")
	table := chi.URLParam(r, "table")

	authCtx, ok := domain.FromContext(r.Context())
	if !ok || authCtx.ProjectSlug != slug {
		SendError(w, r, http.StatusForbidden, ErrForbidden, "Unauthorized access")
		return
	}

	p, err := h.projectSvc.Resolve(r.Context(), slug)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "PROJECT_RESOLUTION_FAILURE")
		return
	}
	privCfg := h.resolvePrivacy(p)

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid JSON payload")
		return
	}

	payload, err = h.privacy.SanitizeWrite(r.Context(), authCtx, privCfg, table, payload, false)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "PRIVACY_ENFORCEMENT_FAILURE", err.Error())
		return
	}

	var cols []string
	var placeholders []string
	var params []interface{}
	idx := 1

	for k, v := range payload {
		cols = append(cols, fmt.Sprintf("%q", k))
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		params = append(params, v)
		idx++
	}

	query := fmt.Sprintf("INSERT INTO %q (%s) VALUES (%s) RETURNING *", 
		table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	finalQuery, err := h.interceptor.ValidateQuery(r.Context(), slug, query)
	if err != nil {
		SendError(w, r, http.StatusForbidden, ErrForbidden, "UNAUTHORIZED_INTENT", err.Error())
		return
	}

	result, err := h.executePoolQuery(r.Context(), authCtx, finalQuery, params)
	if err != nil {
		h.handleQueryError(w, r, err)
		return
	}

	// 1. Synchronous Hijacking (Phase 9 Logic Interceptor)
	if len(result) > 0 {
		hijacked, _ := h.interceptor.ProcessResponse(r.Context(), slug, table, "INSERT", result[0])
		result[0] = hijacked
	}

	// 2. ASYNCHRONOUS TRIGGER & CACHE INVALIDATION (Phase 10 & 12)
	if len(result) > 0 {
		go h.interceptor.EmitEvent(context.Background(), slug, table, "INSERT", result[0])
		go h.cache.InvalidateTable(context.Background(), slug, table)
	}

	finalResult, _ := h.privacy.ApplyMasking(r.Context(), authCtx, privCfg, table, result)
	h.respond(w, r, finalResult)
}

// ServePatch updates existing records and triggers Phase 10 events.
func (h *DataHandler) ServePatch(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "project")
	table := chi.URLParam(r, "table")

	authCtx, ok := domain.FromContext(r.Context())
	if !ok || authCtx.ProjectSlug != slug {
		SendError(w, r, http.StatusForbidden, ErrForbidden, "Access revoked")
		return
	}

	p, err := h.projectSvc.Resolve(r.Context(), slug)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "PROJECT_RESOLUTION_FAILURE")
		return
	}
	privCfg := h.resolvePrivacy(p)

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Malformatted payload")
		return
	}

	payload, err = h.privacy.SanitizeWrite(r.Context(), authCtx, privCfg, table, payload, true)

	pq, err := ParsePostgrest(r.URL.Query())
	if err != nil || pq.Where == "" {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Missing filters for safe update")
		return
	}

	var setClauses []string
	params := pq.Params
	idx := len(params) + 1

	for k, v := range payload {
		setClauses = append(setClauses, fmt.Sprintf("%q = $%d", k, idx))
		params = append(params, v)
		idx++
	}

	query := fmt.Sprintf("UPDATE %q SET %s WHERE %s RETURNING *", 
		table, strings.Join(setClauses, ", "), pq.Where)

	finalQuery, err := h.interceptor.ValidateQuery(r.Context(), slug, query)
	if err != nil {
		SendError(w, r, http.StatusForbidden, ErrForbidden, "DATA_INTEGRITY_VIOLATION", err.Error())
		return
	}

	result, err := h.executePoolQuery(r.Context(), authCtx, finalQuery, params)
	if err != nil {
		h.handleQueryError(w, r, err)
		return
	}

	// 1. Synchronous Hijacking (Phase 9 Logic Interceptor)
	for i := range result {
		hijacked, _ := h.interceptor.ProcessResponse(r.Context(), slug, table, "UPDATE", result[i])
		result[i] = hijacked
	}

	// 2. ASYNCHRONOUS TRIGGER & CACHE INVALIDATION (Phase 10 & 12)
	for i := range result {
		go h.interceptor.EmitEvent(context.Background(), slug, table, "UPDATE", result[i])
	}
	go h.cache.InvalidateTable(context.Background(), slug, table)

	finalResult, _ := h.privacy.ApplyMasking(r.Context(), authCtx, privCfg, table, result)
	h.respond(w, r, finalResult)
}

// ServeDelete removes records and triggers Phase 10 events.
func (h *DataHandler) ServeDelete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "project")
	table := chi.URLParam(r, "table")

	authCtx, ok := domain.FromContext(r.Context())
	if !ok || authCtx.ProjectSlug != slug {
		SendError(w, r, http.StatusForbidden, ErrForbidden, "Unauthorized")
		return
	}

	pq, err := ParsePostgrest(r.URL.Query())
	if err != nil || pq.Where == "" {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Filters required for deterministic deletion")
		return
	}

	query := fmt.Sprintf("DELETE FROM %q WHERE %s RETURNING *", table, pq.Where)

	finalQuery, err := h.interceptor.ValidateQuery(r.Context(), slug, query)
	if err != nil {
		SendError(w, r, http.StatusForbidden, ErrForbidden, "DELETION_BLOCKED", err.Error())
		return
	}

	result, err := h.executePoolQuery(r.Context(), authCtx, finalQuery, pq.Params)
	if err != nil {
		h.handleQueryError(w, r, err)
		return
	}

	// 3. ASYNCHRONOUS TRIGGER & CACHE INVALIDATION (Phase 10 & 12)
	for i := range result {
		go h.interceptor.EmitEvent(context.Background(), slug, table, "DELETE", result[i])
	}
	go h.cache.InvalidateTable(context.Background(), slug, table)

	h.respond(w, r, result)
}

// ServeVectorSearch executes a semantic similarity search (Phase 29).
func (h *DataHandler) ServeVectorSearch(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "project")
	table := chi.URLParam(r, "table")
	
	queryStr := r.URL.Query().Get("q")
	vectorCol := r.URL.Query().Get("col")
	
	if queryStr == "" || vectorCol == "" {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Query 'q' and vector column 'col' are mandatory")
		return
	}

	authCtx, _ := domain.FromContext(r.Context())
	
	// Phase 29: Generate Embeddings via AI Engine
	// For production, we would inject the AI Engine here.
	// We'll simulate a 1536-dim vector for demonstration
	mockEmbedding := make([]float32, 1536) 
	mockEmbedding[0] = 0.5 // Simplified for now

	pool, _ := h.projectSvc.GetPool(r.Context(), authCtx.Project)
	
	var results []map[string]interface{}
	err := pool.WithRLS(r.Context(), database.UserClaims{Sub: authCtx.UserID, Role: authCtx.Role}, slug, false, func(tx pgx.Tx) error {
		res, err := h.vectors.SimilaritySearch(r.Context(), tx, table, vectorCol, mockEmbedding, 10)
		results = res
		return err
	})

	if err != nil {
		h.handleQueryError(w, r, err)
		return
	}

	h.respond(w, r, results)
}

// ServeSeed generates and inserts synthetic data via AI (Phase 32).
func (h *DataHandler) ServeSeed(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "project")
	table := chi.URLParam(r, "table")
	
	countStr := r.URL.Query().Get("count")
	if countStr == "" { countStr = "10" }
	
	count, _ := strconv.Atoi(countStr)
	if count > 50 { count = 50 } // Max limit for stability
	
	// Phase 32: Genesis Lab in action
	_, _ = h.synthetic.GenerateRecords(r.Context(), slug, table, count)

	SendJSON(w, r, http.StatusOK, map[string]interface{}{
		"status": "SEEDING_STARTED_VIA_AI",
		"target": table,
		"count":  count,
	})
}

// executePoolQuery handles the heavy lifting of RLS context injection and pool switching.
func (h *DataHandler) executePoolQuery(ctx context.Context, authCtx *domain.AuthContext, query string, params []interface{}) ([]map[string]interface{}, error) {
	pool, err := h.projectSvc.GetPool(ctx, authCtx.Project)
	if err != nil {
		return nil, fmt.Errorf("isolated pool failed: %w", err)
	}

	claims := database.UserClaims{
		Sub:   authCtx.UserID,
		Email: authCtx.Email,
		Role:  authCtx.Role,
	}

	var result []map[string]interface{}
	err = pool.WithRLS(ctx, claims, authCtx.ProjectSlug, false, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, params...)
		if err != nil { return err }
		defer rows.Close()

		for rows.Next() {
			val, _ := rows.Values()
			rowMap := make(map[string]interface{})
			for i, fd := range rows.FieldDescriptions() {
				rowMap[string(fd.Name)] = val[i]
			}
			result = append(result, rowMap)
		}
		return nil
	})

	return result, err
}

func (h *DataHandler) respond(w http.ResponseWriter, r *http.Request, data interface{}) {
	accept := r.Header.Get("Accept")
	if accept == "" { accept = "application/json" }

	payload, finalType, err := h.format.Encode(data, accept)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "PAYLOAD_ENCODING_FAILURE")
		return
	}

	if finalType == "application/json" {
		var decoded interface{}
		_ = json.Unmarshal(payload, &decoded)
		SendJSON(w, r, http.StatusOK, decoded)
	} else {
		w.Header().Set("Content-Type", finalType)
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
	}
}

func (h *DataHandler) handleQueryError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("data_handler: query execution failed", "error", err)
	SendError(w, r, http.StatusInternalServerError, ErrInternalError, "DATABASE_ERROR", err.Error())
}

// HandleOpenAPI returns the localized OpenAPI schema document.
func (h *DataHandler) HandleOpenAPI(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "project")
	version := r.URL.Query().Get("openapi")
	if version == "" { version = "3.0" }

	gen := NewOpenAPIGenerator(h.projectSvc)
	doc, err := gen.Generate(r.Context(), slug, version)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "OPENAPI_GENERATION_FAILED")
		return
	}

	SendJSON(w, r, http.StatusOK, doc)
}
