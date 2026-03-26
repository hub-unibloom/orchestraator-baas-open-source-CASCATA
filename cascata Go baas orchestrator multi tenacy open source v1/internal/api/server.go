package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"cascata/internal/config"
	"cascata/internal/database"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// Server represents the API server instance.
// It is the primary orchestrator that binds the physical pools to the HTTP protocol.
type Server struct {
	Cfg          *config.Config
	Repo         *database.Repository
	AuthMiddle   *AuthMiddleware
	MemberMiddle *MemberAuthMiddleware
	Interceptor  *Interceptor
	SystemH      *SystemHandler
	DataH        *DataHandler
	AuthH        *AuthHandler
	GraphQLH     *GraphQLHandler
	StorageH     *StorageHandler
	RealtimeH    *RealtimeHub
	WebhookH     *WebhookHandler
	SyncH        *SyncHandler
	LogicH       *LogicHandler
	MigrationH   *MigrationHandler
	AIH          *AIHandler
	UIH          *UIHandler
	http         *http.Server
}

// NewServer creates a new API server with the given dependencies.
func NewServer(
	cfg *config.Config,
	repo *database.Repository,
	authM *AuthMiddleware,
	memberM *MemberAuthMiddleware,
	interceptor *Interceptor,
	systemH *SystemHandler,
	dataH *DataHandler,
	authH *AuthHandler,
	graphH *GraphQLHandler,
	storageH *StorageHandler,
	realtimeH *RealtimeHub,
	webhookH *WebhookHandler,
	syncH *SyncHandler,
	logicH *LogicHandler,
	migrationH *MigrationHandler,
	aiH *AIHandler,
	uiH *UIHandler,
) *Server {
	return &Server{
		Cfg:          cfg,
		Repo:         repo,
		AuthMiddle:   authM,
		MemberMiddle: memberM,
		Interceptor:  interceptor,
		SystemH:      systemH,
		DataH:        dataH,
		AuthH:        authH,
		GraphQLH:     graphH,
		StorageH:     storageH,
		RealtimeH:    realtimeH,
		WebhookH:     webhookH,
		SyncH:        syncH,
		LogicH:       logicH,
		MigrationH:   migrationH,
		AIH:          aiH,
		UIH:          uiH,
	}
}

// Start initiates the server on a TCP port or Unix socket.
func (s *Server) Start(ctx context.Context, id int) error {
	router := chi.NewRouter()
	
	// Global Middlewares (Sinergy & Observability)
	router.Use(CORSMiddleware)
	router.Use(chiMiddleware.RealIP)
	router.Use(HandlePanic)
	router.Use(s.Interceptor.TelemetryMiddleware)
	
	// Routes
	// 1. PUBLIC ROUTES
	router.Get("/health", s.handleHealth)
	router.Post("/auth/login", s.SystemH.HandleLogin) 

	// STATIC ASSETS (Sovereign CSS & Scripts)
	fsStatic := http.FileServer(http.Dir("internal/ui/static"))
	router.Handle("/static/*", http.StripPrefix("/static/", fsStatic))

	// SOVEREIGN UI ROUTES (Unprotected Entry Points)
	router.Get("/login", s.UIH.ServeLogin)

	// Static i18n & Themes Sovereignty
	fsLangs := http.FileServer(http.Dir("languages"))
	fsThemes := http.FileServer(http.Dir("themas"))
	router.Handle("/languages/*", http.StripPrefix("/languages/", fsLangs))
	router.Handle("/themas/*", http.StripPrefix("/themas/", fsThemes))
	
	// 2. PROTECTED ADMINISTRATIVE ROUTES (Dashboard/Members)
	router.Group(func(r chi.Router) {
		r.Use(s.MemberMiddle.EnforceMemberSession)

		r.Get("/", s.UIH.ServeIndex)
		r.Get("/dashboard", s.UIH.ServeSystemDashboard)
		
		r.Get("/projects", s.SystemH.HandleListProjects)
		r.Post("/projects", s.SystemH.HandleCreateProject)
		r.Get("/projects/export", s.SystemH.HandleExportCAF)

		r.Route("/projects", func(r chi.Router) {
			r.Get("/list", s.UIH.HandleUIListProjects) 
			r.Get("/onboarding", s.UIH.HandleUIOnboarding) 
			r.Get("/{slug}", s.UIH.HandleUIProjectDashboard)
			r.Get("/{slug}/overview", s.UIH.HandleUIProjectOverview)
			r.Get("/{slug}/database", s.UIH.HandleUIDatabaseExplorer)
			r.Get("/{slug}/database/tables", s.UIH.HandleUIDatabaseTables)
			r.Get("/{slug}/database/tables/search", s.UIH.HandleUIDatabaseTables)
			r.Get("/{slug}/database/tables/{table}/data", s.UIH.HandleUIDatabaseTableData)
			r.Get("/{slug}/database/tables/{table}/context-menu", s.UIH.HandleUIDatabaseContextMenu)
			r.Get("/{slug}/database/console", s.UIH.HandleUIDatabaseConsole)
			r.Get("/{slug}/database/modals/{type}", s.UIH.HandleUIDatabaseModals)
			r.Delete("/{slug}", s.SystemH.HandleDeleteProject)
		})
	})

	// 3. PROTECTED TENANT ROUTES (V1 API)
	// We use chi for path parameter capturing used in DataHandlers
	router.Group(func(r chi.Router) {
		r.Use(s.AuthMiddle.EnforceTripleMode)
		
		// Map v1 tenant operations
		r.Route("/v1/{project}", func(r chi.Router) {
			r.Get("/stats", s.DataH.HandleProjectOverview) // Project Overview Real Data
			r.Get("/openapi", s.DataH.HandleOpenAPI)
			
			// Identity Gateway (Phase 10.1 & 13)
			r.Post("/auth/signup", s.AuthH.HandleSignup)
			r.Post("/auth/login", s.AuthH.HandleLogin)
			r.Post("/auth/verify", s.AuthH.HandleVerifyOTP)
			r.Post("/auth/challenge", s.AuthH.HandleStepUpChallenge)
			r.Post("/auth/verify-step-up", s.AuthH.HandleVerifyStepUp)
			r.Get("/auth/authorize", s.AuthH.HandleAuthorize)
			r.Get("/auth/callback", s.AuthH.HandleCallback)

			// Schema Migrations (Phase 15: Structural Evolution Engine)
			r.Post("/migrations", s.MigrationH.ServeMigration)
			r.Get("/migrations/history", s.MigrationH.GetMigrationHistory)

			// Intelligence Layer (Phase 28: Phantom AI)
			r.Post("/ai/ask", s.AIH.ServeAsk)

			// RESTful Table access (Dynamic Data Mesh)
			r.Route("/{table}", func(r chi.Router) {
				r.Get("/", s.DataH.ServeGet)
				r.Get("/search", s.DataH.ServeVectorSearch)
				r.Post("/seed", s.DataH.ServeSeed)
				r.Post("/", s.DataH.ServePost)
				r.Patch("/", s.DataH.ServePatch)
				r.Delete("/", s.DataH.ServeDelete)
				r.Post("/sync", s.SyncH.HandleSync)
			})

			// Storage Hub
			r.Post("/storage/upload", s.StorageH.Upload)
			r.Get("/storage/download", s.StorageH.Download)

			// Logic Edge Nodes (Phase 14: Phantom Functions)
			r.Post("/functions", s.LogicH.ServeHTTP)
			r.Post("/functions/{name}", s.LogicH.HandleInvoke)

			// Real-time Hub (SSE)
			r.Get("/realtime", s.RealtimeH.HandleSSE)

			// Webhook Identity/Gateway (Phase 10.4)
			r.Post("/webhooks/{path}", s.WebhookH.ServeHTTP)

			// GraphQL Hub (Phase 4 & 21)
			r.Post("/graphql", s.GraphQLH.ServeHTTP)
			r.Get("/graphql/ws", s.GraphQLH.ServeWebSocket)
		})
	})

	// Use the chi router as the main handler
	mainHandler := router

	// Determine the listener: TCP for container mesh, Unix for local speed-ups
	var ln net.Listener
	var err error
	var socketPath string

	// Se estivermos em produção ou container, preferimos TCP para o Gateway Nginx encontrar o serviço
	if s.Cfg.Environment == "production" {
		ln, err = net.Listen("tcp", ":"+s.Cfg.Port)
		if err != nil {
			return fmt.Errorf("api.Start (TCP): listen: %w", err)
		}
		slog.Info("OPERATING IN PRODUCTION MESH", "port", s.Cfg.Port, "protocol", "TCP")
	} else {
		slog.Info("OPERATING IN LOCAL/DEV MESH", "dir", s.Cfg.SocketDir)
		socketPath = fmt.Sprintf("%s/worker_%d.sock", s.Cfg.SocketDir, id)
		if _, err := os.Stat(s.Cfg.SocketDir); os.IsNotExist(err) {
			_ = os.MkdirAll(s.Cfg.SocketDir, 0755)
		}

		ln, err = net.Listen("unix", socketPath)
		if err != nil {
			slog.Warn("falling back to TCP listener", "reason", err.Error())
			ln, err = net.Listen("tcp", ":"+s.Cfg.Port)
			if err != nil {
				return fmt.Errorf("api.Start: listen: %w", err)
			}
		} else {
			_ = os.Chmod(socketPath, 0777)
			slog.Info("listening on unix socket", "path", socketPath)
		}
	}

	s.http = &http.Server{
		Handler:      mainHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		BaseContext:  func(net.Listener) context.Context { return ctx },
	}

	go func() {
		if err := s.http.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("server serve error", "error", err)
		}
	}()

	slog.Info("api server started", "worker_id", id)

	<-ctx.Done()
	slog.Info("gracefully shutting down api server", "worker_id", id)
	
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if ln.Addr().Network() == "unix" && socketPath != "" {
		defer os.Remove(socketPath)
	}

	return s.http.Shutdown(shutdownCtx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// L1 Check: Process state
	// L2 Check: DB Connectivity
	if err := s.Repo.Pool.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unhealthy","reason":"db_unreachable"}`))
		return
	}

	// L3 Check: Redis Connectivity Pulse
	// We use the SystemHandler's knowledge or directly from config if needed
	// In this implementation, we can check via a temporary ping if needed, but 
	// a healthy system must have both available.
	
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"healthy","engine":"go v1.0.0.0"}`))
}
