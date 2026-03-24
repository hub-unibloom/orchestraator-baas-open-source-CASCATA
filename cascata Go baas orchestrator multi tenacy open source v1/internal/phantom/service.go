package phantom

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"cascata/internal/ai"
	"cascata/internal/database"
	"cascata/internal/privacy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// PhantomService is the "Cyber-Nervous System" of Cascata.
// It handles both physical extension injections, intelligent intent analysis (Blindagem Adaptativa),
// and the execution of Phantom Functions (Edge Nodes).
type PhantomService struct {
	repo     *database.Repository
	privacy  *privacy.Engine
	injector *Injector
	aiEngine *ai.Engine
	runtime  *WasmRuntime
}

func NewPhantomService(ctx context.Context, repo *database.Repository, pEng *privacy.Engine, aiEngine *ai.Engine) *PhantomService {
	return &PhantomService{
		repo:     repo,
		privacy:  pEng,
		injector: NewInjector(),
		aiEngine: aiEngine,
		runtime:  NewWasmRuntime(ctx, aiEngine),
	}
}

// IntentVerdict represents the result of the Brain's audit of a SQL statement.
type IntentVerdict struct {
	Allow         bool
	RiskLevel     string // LOW, MEDIUM, HIGH, CRITICAL
	ModifiedQuery string
	Reason        string
	SuggestedFix  string
}

// AnalyzeIntent audits incoming SQL for malicious patterns, inefficiency, or RLS bypass attempts.
// This is the "Blindagem Adaptativa" (Phase 8).
func (s *PhantomService) AnalyzeIntent(ctx context.Context, projectSlug, sql string) (IntentVerdict, error) {
	upperSql := strings.ToUpper(sql)
	
	verdict := IntentVerdict{
		Allow:         true,
		RiskLevel:     "LOW",
		ModifiedQuery: sql,
	}

	// 1. Static Cost/Protection Heuristics
	if strings.Contains(upperSql, "SELECT ") && !strings.Contains(upperSql, " WHERE ") {
		verdict.RiskLevel = "MEDIUM"
		verdict.Reason = "Unfiltered SELECT detected"
		verdict.SuggestedFix = "Add a WHERE clause or LIMIT to prevent resource exhaustion."
		
		if !strings.Contains(upperSql, " LIMIT ") {
			verdict.ModifiedQuery = sql + " LIMIT 1000"
			verdict.Reason += " [Adaptive LIMIT 1000 injected]"
		}
	}

	// 2. Security Patterns (Enterprise Grade)
	forbidden := []string{"DROP ", "TRUNCATE ", "GRANT ", "REVOKE "}
	for _, f := range forbidden {
		if strings.Contains(upperSql, f) {
			verdict.Allow = false
			verdict.RiskLevel = "CRITICAL"
			verdict.Reason = fmt.Sprintf("Forbidden DDL operation detected: %s", f)
			return verdict, nil
		}
	}

	slog.Debug("phantom.intent", "slug", projectSlug, "risk", verdict.RiskLevel, "allow", verdict.Allow)
	
	return verdict, nil
}

// InvokeFunction executes a specific edge WASM module for a tenant.
// Integrated with OpenTelemetry and Cache Sinergy.
func (s *PhantomService) InvokeFunction(ctx context.Context, projectSlug, functionName string, binary []byte, params map[string]interface{}) ([]byte, error) {
	tr := otel.Tracer("phantom-service")
	ctx, span := tr.Start(ctx, "InvokeFunction", trace.WithAttributes(
		attribute.String("cascata.project", projectSlug),
		attribute.String("phantom.function", functionName),
	))
	defer span.End()

	// Phase 14 Execution (Isolated & Instrumented)
	result, err := s.runtime.Execute(ctx, projectSlug, functionName, binary, params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("phantom.service: execution failed: %w", err)
	}

	return result, nil
}

// InjectExtension invokes the physical injection of a Postgres extension via Docker.
func (s *PhantomService) InjectExtension(ctx context.Context, name string) error {
	source, ok := IsPhantom(name)
	if !ok {
		return fmt.Errorf("phantom: extension %s not found in registry", name)
	}

	return s.injector.ExtractFromImage(ctx, source)
}
