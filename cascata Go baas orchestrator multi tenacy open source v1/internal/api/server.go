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
	Cfg      *config.Config
	Repo     *database.Repository
	Middle   *AuthMiddleware
	http     *http.Server
}

// NewServer creates a new API server with the given dependencies.
func NewServer(cfg *config.Config, repo *database.Repository, middle *AuthMiddleware) *Server {
	return &Server{
		Cfg:    cfg,
		Repo:   repo,
		Middle: middle,
	}
}

// Start initiates the server on a TCP port or Unix socket.
func (s *Server) Start(ctx context.Context, id int) error {
	router := http.NewServeMux()
	
	// Routes
	router.HandleFunc("/health", s.handleHealth)
	
	// Protected Area (V1 API)
	// We wrap the entire router or specific paths
	v1Handler := s.Middle.EnforceTripleMode(router)

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
		Handler:      v1Handler,
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
