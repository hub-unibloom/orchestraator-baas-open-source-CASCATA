package main

import (
	"context"
	"flag"
	"fmt"
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
	"cascata/internal/i18n"
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
	
	// Phase 2: Sinergy with Languages
	i18n.Init()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if !*isWorker {
		runPrimary(ctx, cfg)
	} else {
		runWorker(ctx, cfg, *workerID)
	}
}

func runPrimary(ctx context.Context, cfg *config.Config) {
	slog.Info("Cascata v1.0.0.0 Primary: Starting Engine (Synergy Hub)", "env", cfg.Environment)
	
	const defaultWorkers = 2 
	orch := cluster.NewOrchestrator(defaultWorkers)
	
	if err := orch.Start(ctx); err != nil {
		slog.Error("primary: fatal error", "error", err)
	}
	
	slog.Info("Cascata Primary: clean exit")
}

func runWorker(ctx context.Context, cfg *config.Config, id int) {
	slog.Info("Cascata v1.0.0.0 Worker: Initializing", "id", id)

	// --- 1. Infrastructure Layer ---
	otelEngine, err := telemetry.NewTelemetryEngine(ctx, "cascata-orchestrator")
	if err != nil {
		slog.Warn("telemetry: failed to initialize", "error", err)
	} else {
		defer otelEngine.Shutdown(context.Background())
	}

	// --- HEALTH CHECK LOOP (Sovereign Boot Protection) ---
	var repo *database.Repository
	var dfly *redis.Client
	
	slog.Info("worker: awaiting infrastructure synergy...", "id", id)
	for i := 1; i <= 6; i++ {
		success := true
		
		// 1. DB Connect Pulse
		if repo == nil {
			repo, err = database.Connect(ctx, cfg.DatabaseURL)
			if err != nil {
				slog.Warn("worker: database still chilling...", "attempt", i, "err", err)
				success = false
			}
		}

		// 2. Redis Connect Pulse
		if dfly == nil {
			dfly = redis.NewClient(&redis.Options{
				Addr: strings.TrimPrefix(cfg.DflyURL, "redis://"),
			})
			if err := dfly.Ping(ctx).Err(); err != nil {
				slog.Warn("worker: redis still chilling...", "attempt", i, "err", err)
				dfly.Close()
				dfly = nil
				success = false
			}
		}

		if success { break }
		if i == 6 {
			slog.Error("worker: fatal infrastructure failure - target unreachable", "id", id)
			os.Exit(1)
		}
		time.Sleep(5 * time.Second)
	}

	defer repo.Close()
	defer dfly.Close()

	vaultSvc, err := vault.NewVaultService(cfg.VaultAddr, cfg.VaultToken)
	if err != nil { slog.Warn("worker: vault unreachable", "error", err) }
	
	// --- 2. Security & Strategy (Base Services) ---
	auditService := service.NewAuditService(repo)
	poolManager := database.NewTenantPoolManager(repo, otelEngine, auditService)
	go poolManager.CleanupTask(ctx, 5*time.Minute)
	go poolManager.CheckPoolHealth(ctx)

	transitSvc := vault.NewTransitService(vaultSvc.GetClient())
	pEngine := privacy.NewEngine(transitSvc)
	sessionSvc := auth.NewSessionManager(cfg.SystemJWTSecret)
	govSvc := auth.NewGovernanceService()
	
	// --- 3. Repository Layer ---
	projectRepo := repository.NewProjectRepository(repo)
	memberRepo := repository.NewMemberRepository(repo)
	tenantRepo := repository.NewTenantRepository(repo)
	resRepo := repository.NewResidentRepository()

	// --- 4. Core Business Logic (Service Mesh) ---
	projectService := service.NewProjectService(projectRepo, poolManager)
	aiEngine := ai.NewEngine(repo, projectService, poolManager)
	phantomSvc := phantom.NewPhantomService(ctx, repo, pEngine, aiEngine)

	// --- 5. Communication & Identity ---
	rlEngine := ratelimit.NewAdaptiveEngine(dfly, repo)
	otpMgr := auth.NewOTPManager(dfly)
	cacheMgr := database.NewCacheManager(dfly)
	postman := communication.NewTrinityPostman(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)

	residentAuthSvc := auth.NewResidentAuthService(projectService, resRepo, sessionSvc, otpMgr, postman, govSvc, auditService, transitSvc)
	externalAuthSvc := auth.NewExternalAuthService(projectService, resRepo, residentAuthSvc)
	systemAuth := auth.NewSystemAuthService(memberRepo, auditService)
	
	// --- 6. Orchestration Engines (Phase 11-15) ---
	syncService := service.NewSyncService(projectService, auditService)
	migrationService := service.NewMigrationService(cfg, projectService, auditService)
	
	realtimeHub := api.NewRealtimeHub(projectService, pEngine, repo)
	workflowEngine := automation.NewWorkflowEngine(repo, projectService, poolManager, postman, aiEngine, realtimeHub, phantomSvc)
	eventQueue := automation.NewEventQueue(dfly)
	webhookService := webhook.NewService(repo, eventQueue, auditService, otelEngine)
	
	go eventQueue.ListenAndServe(ctx, workflowEngine, fmt.Sprintf("worker-%d", id))
	
	dflyAddr := strings.TrimPrefix(cfg.DflyURL, "redis://")
	scheduler := cron.NewScheduler(dflyAddr, workflowEngine, repo, auditService, otelEngine)
	go func() {
		if err := scheduler.Start(); err != nil { slog.Error("worker: scheduler failure", "err", err) }
	}()

	// --- 7. Provisioning & Storage (Phase 2 & 8) ---
	genesisSvc := service.NewGenesisService(repo, projectRepo, tenantRepo, poolManager, vaultSvc, transitSvc, migrationService)
	indexer := storage.NewIndexer(repo)
	storageSvc := storage.NewService(cfg.StoragePath, dfly, repo, indexer)
	
	authService := auth.NewAuthService(projectService, sessionSvc)

	// --- 8. API Layer Dispatch (The Gateway) ---
	interceptor := api.NewInterceptor(repo, projectRepo, phantomSvc, eventQueue, workflowEngine, auditService)
	
	systemH := api.NewSystemHandler(systemAuth, sessionSvc, rlEngine, genesisSvc, projectRepo, storage.NewBackupService(projectService, repo))
	uiH := api.NewUIHandler(systemH)
	
	srv := api.NewServer(
		cfg, 
		repo, 
		api.NewAuthMiddleware(authService, rlEngine), 
		api.NewMemberAuthMiddleware(sessionSvc), 
		interceptor, 
		systemH, 
		api.NewDataHandler(projectService, interceptor, pEngine, cacheMgr, aiEngine), 
		api.NewAuthHandler(residentAuthSvc, externalAuthSvc, projectService),
		api.NewGraphQLHandler(projectService),
		api.NewStorageHandler(storageSvc, storage.NewMediaOptimizer()),
		realtimeHub,
		api.NewWebhookHandler(webhookService),
		api.NewSyncHandler(syncService),
		api.NewLogicHandler(phantomSvc),
		api.NewMigrationHandler(migrationService),
		api.NewAIHandler(aiEngine),
		uiH,
	)
	
	if err := srv.Start(ctx, id); err != nil {
		slog.Error("worker: fatal crash during execution", "id", id, "error", err)
	}

	slog.Info("Cascata Worker: healthy shutdown", "id", id)
}
