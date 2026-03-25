package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"cascata/internal/database"
	"cascata/internal/i18n"
	"cascata/internal/service"
	"cascata/internal/ui/components"
	"cascata/internal/ui/components/database"
	"cascata/internal/ui/layouts"
	"cascata/internal/ui/pages"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// UIHandler manages the high-fidelity management cockpits.
type UIHandler struct {
	projectSvc service.ProjectService
}

func NewUIHandler(projectSvc service.ProjectService) *UIHandler {
	return &UIHandler{
		projectSvc: projectSvc,
	}
}

// HandleUIRoot serves the main sovereign entry point.
func (h *UIHandler) HandleUIRoot(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	title := "Cascata Orchestrator"

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.Dashboard(loc).Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, true, nil)
	ctx := templ.WithChildren(r.Context(), pages.Dashboard(loc))
	if err := component.Render(ctx, w); err != nil {
		slog.Error("ui: failed to render root page", "err", err)
	}
}

// HandleUIProjects renders the global project matrix.
func (h *UIHandler) HandleUIProjects(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	title := "Project Hub"

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.Projects(loc).Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, true, nil)
	ctx := templ.WithChildren(r.Context(), pages.Projects(loc))
	if err := component.Render(ctx, w); err != nil {
		slog.Error("ui: failed to render projects page", "err", err)
	}
}

// HandleUIProjectOverview renders the pulse dashboard for a specific node.
func (h *UIHandler) HandleUIProjectOverview(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	path := r.URL.Path
	slug := strings.TrimPrefix(path, "/system/projects/")
	slug = strings.TrimSuffix(slug, "/overview")

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		_ = pages.ProjectOverview(slug, loc, pages.ProjectUIStats{}).Render(r.Context(), w)
		return
	}

	title := "Node Pulse: " + slug
	w.Header().Set("Content-Type", "text/html")
	component := layouts.Base(title, loc, false, pages.ProjectSubNav(slug))
	ctx := templ.WithChildren(r.Context(), pages.ProjectOverview(slug, loc, pages.ProjectUIStats{}))
	if err := component.Render(ctx, w); err != nil {
		slog.Error("ui: failed to render project overview page", "slug", slug, "err", err)
	}
}

// HandleUIDatabaseAddColumn processes the DDL request to expand a table's structure.
func (h *UIHandler) HandleUIDatabaseAddColumn(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	table := chi.URLParam(r, "table")
	colName := r.FormValue("name")
	colType := r.FormValue("type")
	schema := r.FormValue("schema")
	if schema == "" { schema = "public" }

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

	// 2. Execute Architectural Shift
	if err := pool.AddColumn(r.Context(), schema, table, colName, colType); err != nil {
		slog.Error("ui: structural expansion failed", "table", table, "col", colName, "err", err)
		http.Error(w, "Structural shift failed", http.StatusInternalServerError)
		return
	}

	// 3. Trigger Matrix Refresh (HTMX Sinergy)
	w.Header().Set("HX-Trigger", "matrix-pulse-refresh")
	w.WriteHeader(http.StatusOK)
}

// HandleUIDatabaseExplorer renders the management cockpit for database operations.
func (h *UIHandler) HandleUIDatabaseExplorer(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

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

// HandleUIDatabaseRows returns the data rows for a specific table.
func (h *UIHandler) HandleUIDatabaseRows(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("ui: column fetch failed", "table", table, "err", err)
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
		table := r.URL.Query().Get("table")
		_ = database.DeleteTableConfirm(slug, table, "public").Render(r.Context(), w)
	case "add-column":
		table := r.URL.Query().Get("table")
		// For now we serve the simple confirm or a drawer if we had it.
		// Let's assume we use a simpler modal for now.
		w.Write([]byte(`<div class="p-10 text-white font-black italic">ADD_COLUMN_WORK_IN_PROGRESS</div>`))
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

// HandleUIEdgeFunctions renders the Sovereign Edge & Logic IDE.
func (h *UIHandler) HandleUIEdgeFunctions(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	slug := chi.URLParam(r, "slug")

	// Bring-Up Assets
	assets := []pages.ProjectAsset{
		{Name: "verify_user_pulse", Type: "rpc"},
		{Name: "on_user_created", Type: "trigger"},
		{Name: "core_logic", Type: "folder"},
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

	tables := []string{"profiles", "transactions"}
	rpcs := []string{"calculate_tax", "verify_pulse"}

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

	snapshots := []pages.Snapshot{{Name: "GENESIS", Size: "1.2 GB", CreatedAt: "2024-03-25"}}
	policies := []pages.BackupPolicy{{Name: "DAILY_S3", Provider: "aws_s3", Cron: "0 0 * * *"}}
	history := []pages.BackupHistory{{Type: "FULL", Timestamp: "1d ago", Duration: "12s"}}

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
// HandleUIOnboarding serves the high-fidelity project genesis sequence.
func (h *UIHandler) HandleUIOnboarding(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	w.Header().Set("Content-Type", "text/html")
	_ = pages.OnboardingPage(loc).Render(r.Context(), w)
}

// HandleUIServeLogin serves the identity gateway entry point.
func (h *UIHandler) HandleUIServeLogin(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	w.Header().Set("Content-Type", "text/html")
	_ = pages.LoginPage(loc).Render(r.Context(), w)
}
