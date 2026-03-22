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
)

// Server represents the API server instance.
type Server struct {
	Cfg          *config.Config
	Repo         *Repository
	AuthMiddle   *AuthMiddleware
	MemberMiddle *MemberAuthMiddleware
	SystemH      *SystemHandler
	http         *http.Server
}

// NewServer creates a new API server with the given dependencies.
func NewServer(cfg *config.Config, repo *Repository, authM *AuthMiddleware, memberM *MemberAuthMiddleware, systemH *SystemHandler) *Server {
	return &Server{
		Cfg:          cfg,
		Repo:         repo,
		AuthMiddle:   authM,
		MemberMiddle: memberM,
		SystemH:      systemH,
	}
}

// Start initiates the server on a TCP port or Unix socket.
func (s *Server) Start(ctx context.Context, id int) error {
	router := http.NewServeMux()
	
	// Routes
	// 1. PUBLIC ROUTES
	router.HandleFunc("/health", s.handleHealth)
	router.HandleFunc("/system/auth/login", s.SystemH.HandleLogin) // Public (Rate limited inside handler)
	
	// 2. PROTECTED SYSTEM ROUTES (Dashboard/Members)
	systemMux := http.NewServeMux()
	systemMux.HandleFunc("/system/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message": "List of projects (Mocked)"}`))
	})
	router.Handle("/system/", s.MemberMiddle.EnforceMemberSession(systemMux))

	// 3. PROTECTED TENANT ROUTES (V1 API)
	v1Mux := http.NewServeMux()
	// All V1 tenant operations go here
	v1Mux.HandleFunc("/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message": "Welcome to Tenant API (Mocked)"}`))
	})
	router.Handle("/v1/", s.AuthMiddle.EnforceTripleMode(v1Mux))

	// Use the root router as the main handler
	mainHandler := router

	// Determine the listener: Unix Socket or TCP
	var ln net.Listener
	var err error

	socketPath := fmt.Sprintf("%s/worker_%d.sock", s.Cfg.SocketDir, id)
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
		// Set permissions for Unix Socket so Nginx can access it
		_ = os.Chmod(socketPath, 0777)
		slog.Info("listening on unix socket", "path", socketPath)
	}

	s.http = &http.Server{
		Handler:      mainHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
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

	if ln.Addr().Network() == "unix" {
		defer os.Remove(socketPath)
	}

	return s.http.Shutdown(shutdownCtx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// L1 Check: Process state
	// L2 Check: DB Connectivity
	err := s.Repo.Pool.Ping(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unhealthy","reason":"db"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"healthy","engine":"go v1.0.0.0"}`))
}
