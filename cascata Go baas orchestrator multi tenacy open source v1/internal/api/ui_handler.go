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
	"cascata/internal/service"
	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
)

// UIHandler manages the sovereign web interface rendering with i18n support.
type UIHandler struct {
	SystemH *SystemHandler
	projectSvc *service.ProjectService
}

func NewUIHandler(systemH *SystemHandler, projectSvc *service.ProjectService) *UIHandler {
	return &UIHandler{
		SystemH: systemH,
		projectSvc: projectSvc,
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
	slug := r.URL.Path[len("/system/projects/"):]
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
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.ProjectDashboard(slug, loc))
	if err := component.Render(ctx, w); err != nil {
		slog.Error("ui: failed to render project dashboard page", "slug", slug, "err", err)
	}
}

// HandleUIProjectOverview returns the stats fragment for the cockpit.
func (h *UIHandler) HandleUIProjectOverview(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	path := r.URL.Path
	slug := strings.TrimPrefix(path, "/system/projects/")
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

// HandleUIDatabaseTables returns the list of tables for a specific schema.
func (h *UIHandler) HandleUIDatabaseTables(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	// 1. Resolve Project
	p, err := h.projectSvc.Resolve(r.Context(), slug)
	if err != nil {
		slog.Error("ui: project resolution failed", "slug", slug, "err", err)
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// 2. Resolve Multi-tenant Connection Pool
	pool, err := h.projectSvc.GetPool(r.Context(), p)
	if err != nil {
		slog.Error("ui: tenant pool failure", "slug", slug, "err", err)
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// 3. Introspect Schema (Sovereign discovery)
	tables, err := pool.GetTables(r.Context(), "public")
	if err != nil {
		slog.Error("ui: schema introspection failure", "slug", slug, "err", err)
		http.Error(w, "Failed to list tables", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := database.TableList(slug, tables, loc).Render(r.Context(), w); err != nil {
		slog.Error("ui: failed to render table list", "slug", slug, "err", err)
	}
}

// HandleUIDatabaseExplorer renders the management cockpit for database operations.
func (h *UIHandler) HandleUIDatabaseExplorer(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	path := r.URL.Path
	slug := strings.TrimPrefix(path, "/system/projects/")
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
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
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

// HandleUIDatabaseRows returns the data rows for a specific table.
func (h *UIHandler) HandleUIDatabaseRows(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")
	table := chi.URLParam(r, "table")

	// 1. Resolve Project and Pool (Sovereign isolation)
	p, err := h.projectSvc.Resolve(r.Context(), slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	pool, err := h.projectSvc.GetPool(r.Context(), p)
	if err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// 2. Introspect Schema for Columns (Metadata Pulse)
	cols, err := pool.GetColumns(r.Context(), "public", table)
	if err != nil {
		slog.Error("ui: row fetch failed", "table", table, "err", err)
		http.Error(w, "Failed to inspect table", http.StatusInternalServerError)
		return
	}

	// 3. Fetch Data Rows
	data, err := pool.FetchRows(r.Context(), table, 100)
	if err != nil {
		slog.Error("ui: row fetch failed", "table", table, "err", err)
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := database.TablePanel(slug, "public", table, cols, data, 1, 100).Render(r.Context(), w); err != nil {
		slog.Error("ui: failed to render table panel", "slug", slug, "err", err)
	}
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
	switch modalType {
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
	slug := strings.TrimPrefix(path, "/system/projects/")
	slug = strings.TrimSuffix(slug, "/settings")

	templ.Handler(components.ProjectSettingsModal(slug, loc)).ServeHTTP(w, r)
}

// HandleUIEdgeFunctions renders the Sovereign Edge & Logic IDE.
func (h *UIHandler) HandleUIEdgeFunctions(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	// Bring-Up Assets
	assets := []pages.ProjectAsset{
		{Name: "verify_user_pulse", Type: "rpc"},
		{Name: "on_user_created", Type: "trigger"},
		{Name: "core_logic", Type: "folder"},
		{Name: "process_ledger", Type: "rpc"},
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.EdgeFunctionsPage(slug, assets, loc).Render(r.Context(), w)
		return
	}

	title := "Edge Engine: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.EdgeFunctionsPage(slug, assets, loc))
	_ = component.Render(ctx, w)
}

// HandleUIAPIDocs renders the Sovereign API Gateway & Docs cockpit.
func (h *UIHandler) HandleUIAPIDocs(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.APIDocPage(slug, loc).Render(r.Context(), w)
		return
	}

	title := "API Gateway: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.APIDocPage(slug, loc))
	_ = component.Render(ctx, w)
}

// HandleUIIntelligence renders the Sovereign Neural Core (AI Governance) cockpit.
func (h *UIHandler) HandleUIIntelligence(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	// Bring-Up Metadata
	tables := []string{"profiles", "transactions", "wal_registry", "audit_log", "residents"}
	rpcs := []string{"calculate_tax", "verify_resident_pulse", "drain_cache_pool", "sync_temporal_logs"}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.IntelligencePage(slug, tables, rpcs, loc).Render(r.Context(), w)
		return
	}

	title := "Neural Core: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.IntelligencePage(slug, tables, rpcs, loc))
	_ = component.Render(ctx, w)
}

// HandleUIBackups renders the Sovereign Time Machine cockpit.
func (h *UIHandler) HandleUIBackups(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	// Phase 10: Structural Bring-Up Data (Telemetry ready)
	snapshots := []pages.Snapshot{
		{Name: "GENESIS_RESTORE_POINT", Size: "1.24 GB", CreatedAt: "2024-03-25 04:00:21"},
		{Name: "POST_SCHEMA_UPGRADE_15", Size: "1.26 GB", CreatedAt: "2024-03-25 09:12:44"},
	}
	policies := []pages.BackupPolicy{
		{Name: "PRIMARY_GDRIVE_DRAIN", Provider: "gdrive", Cron: "0 4 * * *"},
		{Name: "S3_DISASTER_NODE", Provider: "aws_s3", Cron: "0 0 * * 0"},
	}
	history := []pages.BackupHistory{
		{Type: "FULL_CAF_UPLOAD", Timestamp: "12m ago", Duration: "45.2s"},
		{Type: "SNAPSHOT_PULSE_GEN", Timestamp: "4h ago", Duration: "1.8s"},
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.BackupsPage(slug, snapshots, policies, history, loc).Render(r.Context(), w)
		return
	}

	title := "Time Machine: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.BackupsPage(slug, snapshots, policies, history, loc))
	_ = component.Render(ctx, w)
}

// HandleUICommCenter renders the Sovereign Communication Hub.
func (h *UIHandler) HandleUICommCenter(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.CommCenter(slug, loc).Render(r.Context(), w)
		return
	}

	title := "Comm Center: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.CommCenter(slug, loc))
	_ = component.Render(ctx, w)
}

// HandleUICommSubSection serves the inner fragments of the Comm Center.
func (h *UIHandler) HandleUICommSubSection(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	subType := chi.URLParam(r, "type")

	w.Header().Set("Content-Type", "text/html")
	switch subType {
	case "push":
		_ = pages.PushEngine(slug).Render(r.Context(), w)
	case "connectors":
		// Placeholder for connectors grid
		w.Write([]byte(`<div class="p-20 text-center text-[10px] font-black uppercase tracking-[0.5em] text-content-muted italic opacity-20">CONNECT_HUBS_AWAITING_PULSE...</div>`))
	case "config":
		// Placeholder for comm config
		w.Write([]byte(`<div class="p-20 text-center text-[10px] font-black uppercase tracking-[0.5em] text-content-muted italic opacity-20">CONFIG_PARAMETERS_OFFLINE...</div>`))
	}
}

// HandleUISecurityLab renders the Sovereign Authentication & RLS Designer.
func (h *UIHandler) HandleUISecurityLab(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.SecurityLab(slug, loc).Render(r.Context(), w)
		return
	}

	title := "Security Lab: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.SecurityLab(slug, loc))
	_ = component.Render(ctx, w)
}

// HandleUIStorageExplorer renders the Sovereign File & Blob Orchestrator.
func (h *UIHandler) HandleUIStorageExplorer(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.StorageExplorer(slug, loc).Render(r.Context(), w)
		return
	}

	title := "Storage Explorer: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.StorageExplorer(slug, loc))
	_ = component.Render(ctx, w)
}

// HandleUILedgeLedger renders the Sovereign Event & Log Trace Orchestrator.
func (h *UIHandler) HandleUILedgeLedger(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.LedgeLedger(slug, loc).Render(r.Context(), w)
		return
	}

	title := "Ledge Ledger: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.LedgeLedger(slug, loc))
	_ = component.Render(ctx, w)
}

// HandleUISettings renders the Sovereign Node Configuration page.
func (h *UIHandler) HandleUISettings(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.ProjectSettingsPage(slug, loc).Render(r.Context(), w)
		return
	}

	title := "Node Settings: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.ProjectSettingsPage(slug, loc))
	_ = component.Render(ctx, w)
}
