package api

import (
	"net/http"
	"cascata/internal/ui/layouts"
	"cascata/internal/ui/pages"
	"cascata/internal/i18n"
	"cascata/internal/ui/components"
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

// ServeIndex renders the main entry point with the correct localization.
func (h *UIHandler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	
	// Base Layout renders everything with the localizer context
	component := layouts.Base(i18n.T(loc, "dashboard_title"), loc)
	templ.Handler(component).ServeHTTP(w, r)
}

// ServeSystemDashboard returns the dashboard fragment.
func (h *UIHandler) ServeSystemDashboard(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	if r.Header.Get("HX-Request") == "true" {
		templ.Handler(pages.Dashboard(loc)).ServeHTTP(w, r)
		return
	}
	h.ServeIndex(w, r)
}

// HandleUIListProjects returns a fragment containing the grid of project cards.
func (h *UIHandler) HandleUIListProjects(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	dbProjects, err := h.SystemH.Repo.ListTenants(r.Context())
	if err != nil {
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
	slug := r.URL.Query().Get("slug") // In a real Chi router, this would be chi.URLParam(r, "slug")
	if slug == "" {
		slug = r.URL.Path[len("/system/projects/"):]
	}

	if r.Header.Get("HX-Request") == "true" {
		templ.Handler(pages.ProjectDashboard(slug, loc)).ServeHTTP(w, r)
		return
	}
	h.ServeIndex(w, r)
}

// HandleUIProjectOverview returns the stats fragment for the cockpit.
func (h *UIHandler) HandleUIProjectOverview(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	
	// Extraction of Project Slug from path: /system/projects/{slug}/overview
	path := r.URL.Path
	slug := strings.TrimPrefix(path, "/system/projects/")
	slug = strings.TrimSuffix(slug, "/overview")

	// 1. Fetch real metrics from DB
	dbTenants, _ := h.SystemH.Repo.ListTenants(r.Context())
	var currentTenant *database.Tenant
	for _, t := range dbTenants {
		if t.Slug == slug {
			currentTenant = &t
			break
		}
	}

	if currentTenant == nil {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	// 2. Prepare Stats Model (Simplified for initial migration)
	stats := pages.ProjectStats{
		Status:           currentTenant.Status,
		TotalUsers:       currentTenant.MaxUsers, // Using max users as initial proxy
		TotalTables:      0,                      // To be detailed in Phase 2
		SchemaSizeBytes:  int64(currentTenant.MaxStorageMB) * 1024 * 1024 / 2, // Proxy usage for UI view
		TrafficRate:      "0.0 KB/s",
		TableNames:       []string{},             // Empty for now (Genesis Phase)
	}

	// 3. Render the fragment
	templ.Handler(pages.ProjectOverview(slug, loc, stats)).ServeHTTP(w, r)
}

// HandleUIProjectSettings returns the settings modal fragment for a project.
func (h *UIHandler) HandleUIProjectSettings(w http.ResponseWriter, r *http.Request) {
	loc := i18n.GetLocalizer(r)
	
	// Extraction of Project Slug from path: /system/projects/{slug}/settings
	path := r.URL.Path
	slug := strings.TrimPrefix(path, "/system/projects/")
	slug = strings.TrimSuffix(slug, "/settings")

	templ.Handler(components.ProjectSettingsModal(slug, loc)).ServeHTTP(w, r)
}
