package api

import (
	"log/slog"
	"net/http"
	"strings"
	"strconv"
	"cascata/internal/ui/layouts"
	"cascata/internal/ui/pages"
	"cascata/internal/i18n"
	"cascata/internal/ui/components"
	"cascata/internal/ui/components/database"
	"cascata/internal/domain"
	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"regexp"
	"fmt"
	"encoding/json"
)

type CreateTableRequest struct {
	TableName   string `json:"tableName"`
	Description string `json:"description"`
	EnableRLS   bool   `json:"enableRLS"`
	McpEnabled  bool   `json:"mcpEnabled"`
	McpPerms    struct {
		R bool `json:"r"`
		C bool `json:"c"`
		U bool `json:"u"`
		D bool `json:"d"`
	} `json:"mcpPerms"`
	Columns []struct {
		ID           interface{} `json:"id"`
		Name         string      `json:"name"`
		Type         string      `json:"type"`
		DefaultValue string      `json:"defaultValue"`
		Nullable     bool        `json:"nullable"`
		PK           bool        `json:"pk"`
		Unique       bool        `json:"unique"`
		Array        bool        `json:"array"`
		Identity     *string     `json:"identity"`
		Lock         *string     `json:"lock"`
		Format       *string     `json:"format"`
	} `json:"columns"`
}

// UIHandler manages the sovereign web interface rendering with i18n support.
type UIHandler struct {
	SystemH *SystemHandler
}

func NewUIHandler(systemH *SystemHandler) *UIHandler {
	return &UIHandler{
		SystemH: systemH,
	}
}

// ServeIndex renders the main entry point (Authenticated Dashboard).
func (h *UIHandler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	title := i18n.T(loc, "dashboard_title")
	
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, nil)
	
	// Canonical Templ Child Injection in Go code
	ctx := templ.WithChildren(r.Context(), pages.Dashboard(loc))
	if err := component.Render(ctx, w); err != nil {
		slog.Error("ui: failed to render index", "err", err)
	}
}

// ServeLogin renders the sovereign authentication portal.
func (h *UIHandler) ServeLogin(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	w.Header().Set("Content-Type", "text/html")
	if err := pages.Login(loc).Render(r.Context(), w); err != nil {
		slog.Error("ui: failed to render login", "err", err)
	}
}

// ServeSystemDashboard returns the dashboard fragment for HTMX requests.
func (h *UIHandler) ServeSystemDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		loc := i18n.GetLocalizer(r)
		w.Header().Set("Content-Type", "text/html")
		if err := pages.Dashboard(loc).Render(r.Context(), w); err != nil {
			slog.Error("ui: failed to render dashboard fragment", "err", err)
		}
		return
	}
	h.ServeIndex(w, r)
}

// HandleUIListProjects returns a fragment containing the grid of project cards.
func (h *UIHandler) HandleUIListProjects(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	dbProjects, err := h.SystemH.ProjectRepo.List(r.Context())
	if err != nil {
		slog.Error("ui: failed to list projects", "err", err)
		http.Error(w, "Failed to fetch projects", http.StatusInternalServerError)
		return
	}

	var uiProjects []components.Project
	for _, p := range dbProjects {
		uiProjects = append(uiProjects, components.Project{
			ID:           p.ID,
			Name:         p.Name,
			Slug:         p.Slug,
			Region:       p.Region,
			Status:       p.Status,
			MaxUsers:     p.MaxUsers,
			MaxConns:     p.MaxConns,
			MaxStorageMB: p.MaxStorageMB,
		})
	}

	for _, p := range uiProjects {
		templ.Handler(components.ProjectCard(p, loc)).ServeHTTP(w, r)
	}

	if len(uiProjects) == 0 {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<div class="col-span-full py-24 text-center border-2 border-dashed border-white/5 rounded-[48px] opacity-40 bg-surface-raised/20">
			<p class="font-black uppercase tracking-[0.4em] text-[10px]">` + i18n.T(loc, "waiting_tenants") + `</p>
		</div>`))
	}
}

// HandleUIOnboarding returns the onboarding modal fragment.
func (h *UIHandler) HandleUIOnboarding(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	templ.Handler(components.OnboardingModal(loc)).ServeHTTP(w, r)
}

// HandleUIProjectDashboard renders the main cockpit page for a tenant.
func (h *UIHandler) HandleUIProjectDashboard(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := r.URL.Path[len("/projects/"):]
	if strings.Contains(slug, "/") {
		slug = slug[:strings.Index(slug, "/")]
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		if err := pages.ProjectDashboard(slug, loc).Render(r.Context(), w); err != nil {
			slog.Error("ui: failed to render project dashboard fragment", "slug", slug, "err", err)
		}
		return
	}

	// Full Page Reload Synergy (Canonical Render)
	title := "Project: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug, "overview"))
	ctx := templ.WithChildren(r.Context(), pages.ProjectDashboard(slug, loc))
	if err := component.Render(ctx, w); err != nil {
		slog.Error("ui: failed to render project dashboard page", "slug", slug, "err", err)
	}
}

// HandleUIProjectOverview returns the stats fragment for the cockpit.
func (h *UIHandler) HandleUIProjectOverview(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	path := r.URL.Path
	slug := strings.TrimPrefix(path, "/projects/")
	slug = strings.TrimSuffix(slug, "/overview")

	dbProjects, err := h.SystemH.ProjectRepo.List(r.Context())
	if err != nil {
		slog.Error("ui: failed to list tenants for overview", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	var currentTenant *domain.Project
	for _, t := range dbProjects {
		if t.Slug == slug {
			currentTenant = t
			break
		}
	}

	if currentTenant == nil {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	stats := pages.ProjectUIStats{
		Status:           currentTenant.Status,
		TotalUsers:       currentTenant.MaxUsers,
		TotalTables:      0,
		SchemaSizeBytes:  currentTenant.MaxStorageMB * 1024 * 1024 / 2, 
		TrafficRate:      "0.0 KB/s",
		TableNames:       []string{},
	}

	templ.Handler(pages.ProjectOverview(slug, loc, stats)).ServeHTTP(w, r)
}

// HandleUIDatabaseExplorer renders the management cockpit for database operations.
func (h *UIHandler) HandleUIDatabaseExplorer(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	path := r.URL.Path
	slug := strings.TrimPrefix(path, "/projects/")
	slug = strings.TrimSuffix(slug, "/database")

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		if err := pages.DatabaseExplorer(slug, loc).Render(r.Context(), w); err != nil {
			slog.Error("ui: failed to render database explorer fragment", "slug", slug, "err", err)
		}
		return
	}

	// Full Page Reload
	title := "Database Explorer: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug, "database"))
	ctx := templ.WithChildren(r.Context(), pages.DatabaseExplorer(slug, loc))
	if err := component.Render(ctx, w); err != nil {
		slog.Error("ui: failed to render database explorer page", "slug", slug, "err", err)
	}
}

// HandleUIDatabaseTables returns the list of tables for a specific schema.
func (h *UIHandler) HandleUIDatabaseTables(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	schema := r.URL.Query().Get("schema")
	if schema == "" { schema = "public" }

	// Mocking tables for initial UI bring-up (matching Sidebar expectations)
	type Table struct { Name string; IsCore bool }
	tables := []Table{
		{Name: "residents", IsCore: true},
		{Name: "auth_audit", IsCore: false},
		{Name: "wal_registry", IsCore: true},
	}

	w.Header().Set("Content-Type", "text/html")
	for _, t := range tables {
		_ = database.TableItem(slug, schema, t.Name, false, t.IsCore).Render(r.Context(), w)
	}
}

// HandleUIDatabaseTableData renders the full Data Mesh for a specific table.
func (h *UIHandler) HandleUIDatabaseTableData(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	table := chi.URLParam(r, "table")
	schema := r.URL.Query().Get("schema")

	// Pre-requisites for the Grid Mesh
	columns := []string{"id", "email", "status", "created_at", "last_pulse"}
	data := []map[string]interface{}{
		{"id": 1, "email": "admin@cascata.io", "status": "verified", "created_at": "2024-03-25", "last_pulse": "1s ago"},
		{"id": 2, "email": "node_alpha@mesh.local", "status": "active", "created_at": "2024-03-24", "last_pulse": "12ms ago"},
	}

	w.Header().Set("Content-Type", "text/html")
	_ = database.TablePanel(slug, schema, table, columns, data, 1, 100).Render(r.Context(), w)
}

// HandleUIDatabaseConsole serves the Sovereign SQL Authority Terminal.
func (h *UIHandler) HandleUIDatabaseConsole(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	loc := i18n.GetLocalizer(r)
	
	w.Header().Set("Content-Type", "text/html")
	_ = database.SqlConsole(slug, loc).Render(r.Context(), w)
}

// HandleUIDatabaseModals serves various management modals (Extensions, Delete, etc).
func (h *UIHandler) HandleUIDatabaseModals(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	modalType := chi.URLParam(r, "type")

	w.Header().Set("Content-Type", "text/html")
	loc := i18n.GetLocalizer(r)
	switch modalType {
	case "create-table":
		_ = database.TableCreator(slug, loc).Render(r.Context(), w)
	case "extensions":
		installed := []string{"pgcrypto", "uuid-ossp"}
		available := []string{"pgcrypto", "uuid-ossp", "pg_vector", "postgis", "pg_cron", "pg_audit"}
		_ = database.ExtensionsModal(slug, installed, available).Render(r.Context(), w)
	case "delete-table":
		// Serves confirm delete modal (placeholder logic)
	}
}

// HandleUIDatabaseContextMenu serves the context menu portal.
func (h *UIHandler) HandleUIDatabaseContextMenu(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	table := chi.URLParam(r, "table")
	schema := r.URL.Query().Get("schema")
	
	xStr := r.URL.Query().Get("x")
	yStr := r.URL.Query().Get("y")
	
	x, _ := strconv.Atoi(xStr)
	y, _ := strconv.Atoi(yStr)

	w.Header().Set("Content-Type", "text/html")
	_ = database.TableContextMenu(slug, schema, table, x, y).Render(r.Context(), w)
}

// HandleUIProjectSettings returns the settings modal fragment for a project.
func (h *UIHandler) HandleUIProjectSettings(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	path := r.URL.Path
	slug := strings.TrimPrefix(path, "/projects/")
	slug = strings.TrimSuffix(slug, "/settings")

	templ.Handler(components.ProjectSettingsModal(slug, loc)).ServeHTTP(w, r)
}
// HandleUIDatabaseCreateTable births a new table from a UI blueprint.
func (h *UIHandler) HandleUIDatabaseCreateTable(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var req CreateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("ui: failed to decode create table request", "err", err)
		http.Error(w, "Invalid blueprint body", http.StatusBadRequest)
		return
	}

	sql := h.generateCreateTableSQL(req, slug)

	// Execute SQL in the context of the project schema
	// Note: In a real production system, we would use the tenant's specific pool
    // and potentially wrap it in a transaction.
	_, err := h.SystemH.ProjectRepo.Repo.Pool.Exec(r.Context(), sql)
	if err != nil {
		slog.Error("ui: failed to execute create table SQL", "slug", slug, "err", err, "sql", sql)
		// Return an error feedback (toast/alert) via HTMX
		w.Header().Set("HX-Retarget", "#database-modal-root")
		fmt.Fprintf(w, `<div class="bg-rose-500/90 text-white p-8 rounded-2xl animate-pulse font-black uppercase text-[10px] tracking-widest border border-rose-400">ARCHITECTURAL_FAILURE: %v</div>`, err)
		return
	}

	// Success: Trigger a refresh of the table list and show success pulse
	w.Header().Set("HX-Trigger", "tableCreated")
    fmt.Fprintf(w, `<div class="bg-emerald-500/90 text-white p-8 rounded-3xl animate-fade-in font-black uppercase text-[10px] tracking-widest border border-emerald-400 flex items-center gap-4 shadow-glow">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="4"><path d="M20 6L9 17l-5-5"/></svg>
        ENTITY_STRIKE_SUCCESS: %s
        <script>setTimeout(() => document.getElementById('table-creator-drawer').remove(), 1500)</script>
    </div>`, req.TableName)
}

func (h *UIHandler) generateCreateTableSQL(req CreateTableRequest, slug string) string {
	tableName := h.SanitizeName(req.TableName)
	schema := slug 
	if schema == "" { schema = "public" }

	var colDefs []string
	for _, c := range req.Columns {
		name := h.SanitizeName(c.Name)
		typeStr := c.Type
		if c.Array { typeStr += "[]" }

		var constraints []string
		if c.PK { constraints = append(constraints, "PRIMARY KEY") }
		if !c.Nullable && !c.PK { constraints = append(constraints, "NOT NULL") }
		if c.Unique && !c.PK { constraints = append(constraints, "UNIQUE") }

		if c.Identity != nil && *c.Identity != "" {
			gen := "ALWAYS"
			if *c.Identity == "by_default" { gen = "BY DEFAULT" }
			constraints = append(constraints, fmt.Sprintf("GENERATED %s AS IDENTITY", gen))
		} else if c.DefaultValue != "" {
			// Basic default value formatting
			val := c.DefaultValue
            if !strings.Contains(val, "(") && !strings.Contains(val, "::") && val != "true" && val != "false" {
                val = "'" + strings.ReplaceAll(val, "'", "''") + "'"
            }
			constraints = append(constraints, fmt.Sprintf("DEFAULT %s", val))
		}

		colDefs = append(colDefs, fmt.Sprintf("    %s %s %s", name, typeStr, strings.Join(constraints, " ")))
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (\n%s\n);", schema, tableName, strings.Join(colDefs, ",\n"))

	// RLS Deployment
	if req.EnableRLS {
		sql += fmt.Sprintf("\nALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY;", schema, tableName)
	}

	// Comments for Format & Lock (UI Metadata)
	for _, c := range req.Columns {
		meta := ""
		if c.Format != nil && *c.Format != "" {
			meta += "||FORMAT:" + *c.Format
		}
		if c.Lock != nil && *c.Lock != "" {
			meta += "||LOCK:" + *c.Lock
		}
		if meta != "" {
			colName := h.SanitizeName(c.Name)
			sql += fmt.Sprintf("\nCOMMENT ON COLUMN %s.%s.%s IS '%s';", schema, tableName, colName, meta)
		}
	}

	// MCP Governance comment
	mcpFlag := ""
	if req.McpEnabled {
		mcpFlag = "||MCP:"
		if req.McpPerms.R { mcpFlag += "R" }
		if req.McpPerms.C { mcpFlag += "C" }
		if req.McpPerms.U { mcpFlag += "U" }
		if req.McpPerms.D { mcpFlag += "D" }
	}
	if req.Description != "" || mcpFlag != "" {
		sql += fmt.Sprintf("\nCOMMENT ON TABLE %s.%s IS '%s%s';", schema, tableName, strings.ReplaceAll(req.Description, "'", "''"), mcpFlag)
	}

	return sql
}

func (h *UIHandler) SanitizeName(val string) string {
	val = strings.ToLower(val)
	re := regexp.MustCompile(`[^a-z0-9_]`)
	val = re.ReplaceAllString(val, "_")
	if len(val) > 0 && val[0] >= '0' && val[0] <= '9' {
		val = "_" + val
	}
	return val
}
