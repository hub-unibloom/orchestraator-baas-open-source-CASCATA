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

	"cascata/internal/ai"
	"cascata/internal/api"
	"cascata/internal/auth"
	"cascata/internal/automation"
	"cascata/internal/cluster"
	"cascata/internal/communication"
	"cascata/internal/config"
	"cascata/internal/cron"
	"cascata/internal/database"
	"cascata/internal/logger"
	"cascata/internal/phantom"
	"cascata/internal/privacy"
	"cascata/internal/ratelimit"
	"cascata/internal/repository"
	"cascata/internal/service"
	"cascata/internal/storage"
	"cascata/internal/telemetry"
	"cascata/internal/vault"
	"cascata/internal/webhook"
)

func main() {
	isWorker := flag.Bool("worker", false, "run as worker process")
	workerID := flag.Int("id", 0, "worker assigned id")
	flag.Parse()

	cfg := config.Load()
	logger.Init(cfg.LogLevel)

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
	
	const defaultWorkers = 2 
	orch := cluster.NewOrchestrator(defaultWorkers)
	
	if err := orch.Start(ctx); err != nil {
		slog.Error("primary: fatal error", "error", err)
	}
	
	slog.Info("Cascata Primary: clean exit")
}

func runWorker(ctx context.Context, cfg *config.Config, id int) {
	slog.Info("Cascata v1.0.0.0 Worker: Initializing", "id", id)

	// 0. Observability Mesh (Phase 17)
	otelEngine, err := telemetry.NewTelemetryEngine(ctx, "cascata-orchestrator")
	if err != nil {
		slog.Warn("telemetry: failed to initialize, continuing without distributed tracing", "error", err)
	} else {
		defer otelEngine.Shutdown(context.Background())
	}

	// 1. Core Persistence & Cache (Dragonfly)
	repo, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("worker: database bootstrap failed", "id", id, "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	dfly := redis.NewClient(&redis.Options{
		Addr: strings.TrimPrefix(cfg.DflyURL, "redis://"),
	})
	defer dfly.Close()

	// 2. Fundamental Services (Audit, Pool, Vault)
	auditService := service.NewAuditService(repo)
	poolManager := database.NewTenantPoolManager(cfg.DatabaseURL, 5000, otelEngine, auditService)
	go poolManager.CleanupTask(ctx, 5*time.Minute)
	go poolManager.CheckPoolHealth(ctx)

	vaultSvc, err := vault.NewVaultService(cfg.VaultAddr, cfg.VaultToken)
	if err != nil { slog.Warn("worker: vault unreachable", "error", err) }
	
	transitSvc := vault.NewTransitService(vaultSvc.GetClient())
	pEngine := privacy.NewEngine(transitSvc)

	// 3. Repository Layer
	projectRepo := repository.NewProjectRepository(repo)
	memberRepo := repository.NewMemberRepository(repo)
	resRepo := repository.NewResidentRepository()

	// 4. Intelligence & Business Logic (AI, Project, Phantom)
	projectService := service.NewProjectService(projectRepo, poolManager)
	aiEngine := ai.NewEngine(repo, projectService, poolManager)
	
	// Unied Phantom Service (Phase 14 Logic & Security)
	phantomSvc := phantom.NewPhantomService(ctx, repo, pEngine, aiEngine)

	// 5. Orchestration & Communication
	rlEngine := ratelimit.NewAdaptiveEngine(dfly, repo)
	otpMgr := auth.NewOTPManager(dfly)
	cacheMgr := database.NewCacheManager(dfly)
	postman := communication.NewTrinityPostman(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)

	// 6. Tenant-Facing Services
	residentAuthSvc := auth.NewResidentAuthService(projectService, resRepo, sessionSvc, otpMgr, postman)
	externalAuthSvc := auth.NewExternalAuthService(projectService, resRepo, residentAuthSvc)
	
	syncService := service.NewSyncService(poolManager, auditService) // Phase 11 Enriched
	migrationService := service.NewMigrationService(poolManager, auditService) // Phase 15 Enriched
	
	// 7. Automation & Real-time Hub (Sinergy Phase 13/14)
	realtimeHub := api.NewRealtimeHub(projectService, pEngine)
	workflowEngine := automation.NewWorkflowEngine(repo, poolManager, postman, aiEngine, realtimeHub)
	eventQueue := automation.NewEventQueue(dfly)
	webhookService := webhook.NewService(repo, eventQueue, auditService, otelEngine)
	
	go eventQueue.ListenAndServe(ctx, workflowEngine, fmt.Sprintf("worker-%d", id))
	
	dflyAddr := strings.TrimPrefix(cfg.DflyURL, "redis://")
	scheduler := cron.NewScheduler(dflyAddr, workflowEngine, repo, auditService, otelEngine)
	go func() {
		if err := scheduler.Start(); err != nil { slog.Error("worker: scheduler failure", "err", err) }
	}()

	// 8. Provisioning & Storage (Phase 2 & 8)
	genesisSvc := service.NewGenesisService(repo, projectRepo, poolManager, vaultSvc, migrationService)
	indexer := storage.NewIndexer(repo)
	storageSvc := storage.NewService(cfg.StoragePath, dfly, repo, indexer)
	
	authService := auth.NewAuthService(projectService)
	systemAuth := auth.NewSystemAuthService(memberRepo)
	sessionSvc := auth.NewSessionManager(cfg.SystemJWTSecret)

	// 9. API Handlers Initialization (Sinergy Wiring)
	interceptor := api.NewInterceptor(repo, projectRepo, phantomSvc, eventQueue, workflowEngine, auditService)
	
	srv := api.NewServer(
		cfg, 
		repo, 
		api.NewAuthMiddleware(authService, rlEngine), 
		api.NewMemberAuthMiddleware(sessionSvc), 
		interceptor, 
		api.NewSystemHandler(systemAuth, sessionSvc, rlEngine, genesisSvc, projectRepo, storage.NewBackupService(poolManager, repo)), 
		api.NewDataHandler(projectService, interceptor, pEngine, cacheMgr, aiEngine), 
		api.NewAuthHandler(residentAuthSvc, externalAuthSvc, projectService),
		api.NewGraphQLHandler(projectService),
		api.NewStorageHandler(storageSvc, storage.NewMediaOptimizer()),
		api.NewRealtimeHub(projectService, pEngine),
		api.NewWebhookHandler(webhookService),
		api.NewSyncHandler(syncService),
		api.NewLogicHandler(phantomSvc),
		api.NewMigrationHandler(migrationService),
		api.NewAIHandler(aiEngine),
	)
	
	if err := srv.Start(ctx, id); err != nil {
		slog.Error("worker: server lifecycle failure", "id", id, "error", err)
	}

	slog.Info("Cascata Worker: clean exit", "id", id)
}
