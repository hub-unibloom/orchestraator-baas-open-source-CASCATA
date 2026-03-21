package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"cascata/internal/api"
	"cascata/internal/auth"
	"cascata/internal/cluster"
	"cascata/internal/config"
	"cascata/internal/database"
	"cascata/internal/logger"
	"cascata/internal/phantom"
	"cascata/internal/privacy"
	"cascata/internal/repository"
	"cascata/internal/service"
	"cascata/internal/storage"
	"cascata/internal/vault"
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

	// 1. Data Layer
	repo, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("worker: database bootstrap failed", "id", id, "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	// 2. Repository Layer
	projectRepo := repository.NewProjectRepository(repo)
	tenantRepo := repository.NewTenantRepository(repo)

	// 3. Service Layer
	projectService := service.NewProjectService(projectRepo)
	migrationService := service.NewMigrationService(cfg)
	phantomInjector := phantom.NewInjector()
	_ = service.NewExtensionService(repo, phantomInjector)
	vaultService, err := vault.NewVaultService(cfg.VaultAddr, cfg.VaultToken)
	if err != nil {
		slog.Warn("vault disabled: connection failed", "error", err)
	}
	
	transitSvc := vault.NewTransitService(vaultService.Client)
	privacyEngine := privacy.NewEngine(transitSvc)
	_ = privacyEngine // will be used in Dispatcher

	storageIndexer := storage.NewIndexer(repo)
	storageSvc := storage.NewService(storageIndexer, repo, "/cascata/storage")
	_ = storageSvc // will be used in Storage Handler

	_ = service.NewOrchestratorService(projectRepo, tenantRepo, migrationService, vaultService)
	authService := auth.NewAuthService(projectService)
	_ = auth.NewGovernanceService()

	// 4. API Layer (Middleware & Server)
	authMiddleware := api.NewAuthMiddleware(authService)
	srv := api.NewServer(cfg, repo, authMiddleware)
	
	// TODO: Wrap routes in authMiddleware.EnforceTripleMode in api/server.go
	
	if err := srv.Start(ctx, id); err != nil {
		slog.Error("worker: server lifecycle ended with error", "id", id, "error", err)
	}

	slog.Info("Cascata Worker: clean exit", "id", id)
}

