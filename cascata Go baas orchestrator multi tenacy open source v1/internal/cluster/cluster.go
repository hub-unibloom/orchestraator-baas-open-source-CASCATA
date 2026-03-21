package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Orchestrator manages the lifecycle of worker processes.
type Orchestrator struct {
	WorkerCount int
	BinaryPath  string
	workers     map[int]*exec.Cmd
	mutex       sync.Mutex
}

// NewOrchestrator creates a new cluster manager.
func NewOrchestrator(count int) *Orchestrator {
	if count <= 0 {
		count = 1 // Default to at least one worker
	}
	return &Orchestrator{
		WorkerCount: count,
		BinaryPath:  os.Args[0], // Use current binary
		workers:     make(map[int]*exec.Cmd),
	}
}

// Start spawns the worker processes and monitors them.
func (o *Orchestrator) Start(ctx context.Context) error {
	slog.Info("hyper-cluster: starting orchestrator", "workers", o.WorkerCount)

	for i := 0; i < o.WorkerCount; i++ {
		go o.spawnWorker(ctx, i)
	}

	<-ctx.Done()
	slog.Info("hyper-cluster: shutting down orchestrator")
	return o.Shutdown()
}

func (o *Orchestrator) spawnWorker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			slog.Info("hyper-cluster: spawning worker", "id", id)

			cmd := exec.CommandContext(ctx, o.BinaryPath, "-worker", fmt.Sprintf("-id=%d", id))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			o.mutex.Lock()
			o.workers[id] = cmd
			o.mutex.Unlock()

			if err := cmd.Run(); err != nil {
				slog.Error("hyper-cluster: worker exited with error", "id", id, "error", err)
			} else {
				slog.Warn("hyper-cluster: worker exited cleanly", "id", id)
			}

			// Self-healing delay
			time.Sleep(1 * time.Second)
		}
	}
}

// Shutdown kills all active workers.
func (o *Orchestrator) Shutdown() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	for id, cmd := range o.workers {
		slog.Info("hyper-cluster: killing worker", "id", id)
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
	}
	return nil
}

// HandleSignals sets up the standard signal handling for the primary process.
func HandleSignals() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
