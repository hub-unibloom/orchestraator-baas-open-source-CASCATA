package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"cascata/internal/api"
	"cascata/internal/auth"
	"cascata/internal/cluster"
	"cascata/internal/config"
	"cascata/internal/database"
	"cascata/internal/logger"
	"cascata/internal/ratelimit"
	"cascata/internal/repository"
	"cascata/internal/service"
)

func main() {
	// 1. Parsing Flags
	isWorker := flag.Bool("worker", false, "run as worker process")
	workerID := flag.Int("id", 0, "worker assigned id")
	flag.Parse()

	// 2. Load Config & Init Logger
	cfg := config.Load()
	logger.Init(cfg.LogLevel)

	// 3. Signal Handling (Standard Context)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if !*isWorker {
		runPrimary(ctx, cfg)
	} else {
		runWorker(ctx, cfg, *workerID)
	}
}

func runPrimary(ctx context.Context, cfg *config.Config) {
	slog.Info("Cascata v1.0.0.0 Primary: Starting Engine", "env", cfg.Environment)
	
	// Primary manages worker pool
	const defaultWorkers = 2 // Should be tunable
	orch := cluster.NewOrchestrator(defaultWorkers)
	
	if err := orch.Start(ctx); err != nil {
		slog.Error("primary: fatal error", "error", err)
	}
	
	slog.Info("Cascata Primary: clean exit")
}

func runWorker(ctx context.Context, cfg *config.Config, id int) {
	slog.Info("Cascata v1.0.0.0 Worker: Initializing", "id", id)

	// 1. Core Infrastucture (DB & Redis/Dragonfly)
	repo, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("worker: database bootstrap failed", "id", id, "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: strings.TrimPrefix(cfg.RedisURL, "redis://"),
	})
	defer rdb.Close()

	// 2. Repository Layer
	projectRepo := repository.NewProjectRepository(repo)
	tenantRepo := repository.NewTenantRepository(repo)
	memberRepo := repository.NewMemberRepository(repo)

	// 3. Manager/Service Layer
	// Connection Pool Isolation per tenant
	poolManager := database.NewTenantPoolManager(cfg.DatabaseURL, 5000)
	go poolManager.CleanupTask(ctx, 5*time.Minute)

	// Adaptive Rate Limiting & Firewall
	rlEngine := ratelimit.NewAdaptiveEngine(rdb, repo)

	// Auth & Project Services
	projectService := service.NewProjectService(projectRepo, poolManager)
	authService := auth.NewAuthService(projectService)
	systemAuth := auth.NewSystemAuthService(memberRepo)
	sessionSvc := auth.NewSessionManager(cfg.SystemJWTSecret)

	// 4. API Layer (Handlers & Middleware)
	authMiddleware := api.NewAuthMiddleware(authService, rlEngine)
	memberMiddleware := api.NewMemberAuthMiddleware(sessionSvc)
	systemHandler := api.NewSystemHandler(systemAuth, sessionSvc, rlEngine)

	srv := api.NewServer(cfg, repo, authMiddleware, memberMiddleware, systemHandler)
	
	if err := srv.Start(ctx, id); err != nil {
		slog.Error("worker: server lifecycle ended with error", "id", id, "error", err)
	}

	slog.Info("Cascata Worker: clean exit", "id", id)
}

