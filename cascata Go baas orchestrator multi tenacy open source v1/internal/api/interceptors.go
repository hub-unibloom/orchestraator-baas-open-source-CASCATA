package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"cascata/internal/automation"
	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/phantom"
	"cascata/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Interceptor struct {
	repo       *database.Repository
	projRepo   *repository.ProjectRepository
	phantomSvc *phantom.PhantomService
	automation *automation.EventQueue
	wfEngine   *automation.WorkflowEngine
	auditSvc   domain.Auditor
}

func NewInterceptor(repo *database.Repository, projRepo *repository.ProjectRepository, phantom *phantom.PhantomService, automation *automation.EventQueue, wfEngine *automation.WorkflowEngine, audit domain.Auditor) *Interceptor {
	return &Interceptor{
		repo:       repo,
		projRepo:   projRepo,
		phantomSvc: phantom,
		automation: automation,
		wfEngine:   wfEngine,
		auditSvc:   audit,
	}
}

// EmmitEvent asynchronously pushes a database change to the automation engine (Phase 10).
func (i *Interceptor) EmitEvent(ctx context.Context, slug, table, operation string, data map[string]interface{}) {
	event := &domain.AutomationEvent{
		ProjectSlug: slug,
		Table:       table,
		Operation:   operation,
		Data:        data,
		Timestamp:   time.Now(),
	}

	if err := i.automation.Publish(ctx, event); err != nil {
		slog.Error("interceptor: event emission failed", "slug", slug, "table", table, "error", err)
	}

	// 2. Audit Critical Events (Phase 19)
	if operation == "DELETE" || operation == "UPDATE" || operation == "TRUNCATE" {
		// Resolve identity from context (Phase 22 Sinergy)
		id := "ADMIN"
		idType := domain.IdentityAgent
		if auth, ok := domain.FromContext(ctx); ok {
			id = auth.UserID
			if id == "" { id = auth.ProjectSlug }
			idType = auth.IdentityType
		}

		go func() {
			entry := &domain.AuditEntry{
				Project:      slug,
				Table:        table,
				Operation:    operation,
				Payload:      fmt.Sprintf("%v", data), // Audit doesn't mask secrets, it's the Ledger of Truth
				IdentityID:   id,
				IdentityType: idType,
				Timestamp:    time.Now(),
			}
			_ = i.auditSvc.WriteEntry(context.Background(), i.repo.Pool, entry)
		}()
	}
}

// ProcessResponse executes synchronous logic hijacking before the client receives the data (Phase 9 Bridge).
// It searches for "API_INTERCEPT" trigger-type workflows associated with the table/operation.
func (i *Interceptor) ProcessResponse(ctx context.Context, slug, table, operation string, data map[string]interface{}) (map[string]interface{}, error) {
	// 1. Resolve Workflows for this project marked as Interceptors (Phase 10 Metadata)
	// In production, we'd cache these lookups.
	wfs, err := i.projRepo.GetWorkflows(ctx, i.repo.Pool, slug)
	if err != nil {
		return data, nil // Fail open for safety, but log it
	}

	for _, wf := range wfs {
		// Only run if it's an interceptor and matches the context
		if wf.Type != "API_INTERCEPT" || wf.Config["table"] != table || wf.Config["operation"] != operation {
			continue
		}

		slog.Debug("interceptor: hijacking response", "wf_id", wf.ID, "slug", slug, "table", table)

		// 2. Wrap current data in an AutomationEvent context
		event := &domain.AutomationEvent{
			ProjectSlug: slug,
			Table:       table,
			Operation:   operation,
			Data:        data,
		}

		// 3. Execute synchronously (Antifragility Loop)
		res, err := i.wfEngine.ExecuteWorkflow(ctx, &wf, event)
		if err != nil {
			slog.Error("interceptor: hijacking failed", "wf_id", wf.ID, "error", err)
			return data, err
		}

		// Use the $last result from workflow execution as the new data
		if resMap, ok := res.(map[string]interface{}); ok {
			data = resMap
		}
	}

	return data, nil
}

// ValidateQuery prevents malicious SQL patterns before execution (Phase 8 - Blindagem Adaptativa).
func (i *Interceptor) ValidateQuery(ctx context.Context, projectSlug, sql string) (string, error) {
	verdict, err := i.phantomSvc.AnalyzeIntent(ctx, projectSlug, sql)
	if err != nil {
		return "", fmt.Errorf("interceptor: brain failure: %w", err)
	}

	if !verdict.Allow {
		slog.Warn("interceptor: block by phantom AI", "slug", projectSlug, "reason", verdict.Reason)
		return "", fmt.Errorf("Security Block: %s", verdict.Reason)
	}

	return verdict.ModifiedQuery, nil
}

// statusWriter is a wrapper for http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// TelemetryMiddleware provides precision tracking and observability for every request (Phase 17 OTel).
func (i *Interceptor) TelemetryMiddleware(next http.Handler) http.Handler {
	tracer := otel.Tracer("cascata-api")
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// 1. Context Propagation (W3C TraceContext)
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		
		// 2. Start Span
		ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path), 
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.user_agent", r.UserAgent()),
			),
		)
		defer span.End()

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		
		// Serve next with traced context
		next.ServeHTTP(sw, r.WithContext(ctx))
		
		latency := time.Since(start)
		
		// Set Span status
		span.SetAttributes(attribute.Int("http.status_code", sw.status))
		if sw.status >= 400 {
			span.RecordError(fmt.Errorf("request_failed_with_status_%d", sw.status))
		}

		slog.Info("api.request", 
			"trace_id", span.SpanContext().TraceID().String(),
			"method", r.Method, 
			"path", r.URL.Path, 
			"status", sw.status,
			"latency_ms", latency.Milliseconds(),
			"remote_ip", r.RemoteAddr,
		)
	})
}
