package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"cascata/internal/automation"
	"cascata/internal/database"

	"github.com/hibiken/asynq"
)

// Task names for the worker.
const (
	TaskRunAutomation = "automation:run"
	TaskPurgeLogs     = "system:purge_logs"
)

// Scheduler manages the lifecycle of timed events.
type Scheduler struct {
	client    *asynq.Client
	scheduler *asynq.Scheduler
	server    *asynq.Server
	engine    *automation.Engine
	repo      *database.Repository
}

func NewScheduler(redisAddr string, engine *automation.Engine, repo *database.Repository) *Scheduler {
	redisOpt := asynq.RedisClientOpt{Addr: redisAddr}
	
	return &Scheduler{
		client:    asynq.NewClient(redisOpt),
		scheduler: asynq.NewScheduler(redisOpt, nil),
		server:    asynq.NewServer(redisOpt, asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		}),
		engine: engine,
		repo:   repo,
	}
}

// Start starts the worker and the scheduler.
func (s *Scheduler) Start() error {
	// Sync active crons on startup
	if err := s.SyncAll(); err != nil {
		slog.Error("cron: sync failed", "error", err)
	}

	// Register Handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskRunAutomation, s.HandleAutomationTask)
	mux.HandleFunc(TaskPurgeLogs, s.HandlePurgeLogs)

	// Run background processes
	go func() {
		if err := s.scheduler.Run(); err != nil {
			slog.Error("cron: scheduler exited", "error", err)
		}
	}()

	return s.server.Run(mux)
}

// SyncAll loads all active cron-type automations into the scheduler.
func (s *Scheduler) SyncAll() error {
	ctx := context.Background()
	sql := `
		SELECT id, project_slug, trigger_config, nodes 
		FROM system.automations 
		WHERE is_active = true AND trigger_type = 'CRON'
	`
	rows, err := s.repo.Pool.Query(ctx, sql)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, slug string
		var configIn, nodesIn []byte
		if err := rows.Scan(&id, &slug, &configIn, &nodesIn); err != nil {
			return err
		}

		var config map[string]interface{}
		var nodes []automation.Node
		json.Unmarshal(configIn, &config)
		json.Unmarshal(nodesIn, &nodes)

		cron, _ := config["cron"].(string)
		if cron == "" { continue }

		var metadataIn []byte
		err := s.repo.Pool.QueryRow(ctx, "SELECT metadata FROM system.projects WHERE slug = $1", slug).Scan(&metadataIn)
		if err == nil {
			var m map[string]interface{}
			json.Unmarshal(metadataIn, &m)
			if tz, ok := m["timezone"].(string); ok && tz != "" {
				tzStr = tz
			}
		}

		s.ScheduleAuto(id, slug, cron)
	}
	return nil
}

// ScheduleAuto adds/updates a repeatable job for an automation.
func (s *Scheduler) ScheduleAuto(id, slug, cron string) {
	// TODO: Use the timezone string when registering in asynq.
	// We'll wrap the spec with the TZ if needed or set it in asynq.Options.
	payload, _ := json.Marshal(map[string]string{
		"automation_id": id,
		"project_slug":  slug,
	})

	task := asynq.NewTask(TaskRunAutomation, payload, asynq.TaskID(id))
	_, err := s.scheduler.Register(cron, task)
	
	if err != nil {
		slog.Error("cron: registration failed", "id", id, "error", err)
	} else {
		slog.Info("cron: scheduled", "id", id, "cron", cron)
	}
}

// HandleAutomationTask processes a single automation run triggered by time.
func (s *Scheduler) HandleAutomationTask(ctx context.Context, t *asynq.Task) error {
	var p map[string]string
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	id := p["automation_id"]
	slug := p["project_slug"]

	// 1. Fetch nodes (AOT concept: we could have pre-compiled these)
	var nodesRaw []byte
	err := s.repo.Pool.QueryRow(ctx, "SELECT nodes FROM system.automations WHERE id = $1", id).Scan(&nodesRaw)
	if err != nil { return err }

	var nodes []automation.Node
	json.Unmarshal(nodesRaw, &nodes)

	// 2. Run Engine
	_, err = s.engine.Run(ctx, nodes, map[string]interface{}{"source": "cron"}, id, slug)
	return err
}

// HandlePurgeLogs implements Phase 11 auto-rotation.
func (s *Scheduler) HandlePurgeLogs(ctx context.Context, t *asynq.Task) error {
	// Purge logic: logs older than X days.
	const sql = `DELETE FROM system.automation_runs WHERE created_at < NOW() - INTERVAL '30 days'`
	_, err := s.repo.Pool.Exec(ctx, sql)
	return err
}
