package api

import (
	"log/slog"
	"net/http"
	"strings"
	"cascata/internal/ui/layouts"
	"cascata/internal/ui/pages"
	"cascata/internal/i18n"
	"cascata/internal/ui/components"
	"cascata/internal/domain"
	"github.com/a-h/templ"
)

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
	// Wrap Dashboard inside Base layout
	title := i18n.T(loc, "dashboard_title")
	component := layouts.Base(title, loc)
	
	// Inject pages.Dashboard as children of layout.Base
	w.Header().Set("Content-Type", "text/html")
	templ.Handler(component, templ.WithChildren(r.Context(), pages.Dashboard(loc))).ServeHTTP(w, r)
}

// ServeLogin renders the sovereign authentication portal.
func (h *UIHandler) ServeLogin(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	templ.Handler(pages.Login(loc)).ServeHTTP(w, r)
}

// ServeSystemDashboard returns the dashboard fragment for HTMX requests.
func (h *UIHandler) ServeSystemDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		loc := i18n.GetLocalizer(r)
		templ.Handler(pages.Dashboard(loc)).ServeHTTP(w, r)
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
		templ.Handler(pages.ProjectDashboard(slug, loc)).ServeHTTP(w, r)
		return
	}

	// Full Page Reload Synergy
	title := "Project: " + slug
	component := layouts.Base(title, loc)
	w.Header().Set("Content-Type", "text/html")
	templ.Handler(component, templ.WithChildren(r.Context(), pages.ProjectDashboard(slug, loc))).ServeHTTP(w, r)
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

// HandleUIProjectSettings returns the settings modal fragment for a project.
func (h *UIHandler) HandleUIProjectSettings(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	path := r.URL.Path
	slug := strings.TrimPrefix(path, "/system/projects/")
	slug = strings.TrimSuffix(slug, "/settings")

	templ.Handler(components.ProjectSettingsModal(slug, loc)).ServeHTTP(w, r)
}
