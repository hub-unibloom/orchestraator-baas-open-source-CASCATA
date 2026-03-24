package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// TelemetryEngine manages the OpenTelemetry lifecycle (Phase 17).
// It provides distributed tracing to ensure every request is observable across the cluster.
type TelemetryEngine struct {
	tp *sdktrace.TracerProvider
}

// NewTelemetryEngine initializes the global tracer and propagator.
func NewTelemetryEngine(ctx context.Context, serviceName string) (*TelemetryEngine, error) {
	slog.Info("telemetry: initializing tracing mesh", "service", serviceName)

	// 1. Exporter: Stdout for local/debug, OTLP for production (Phase 17)
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("telemetry: failed to create exporter: %w", err)
	}

	// 2. Resource configuration
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: failed to create resource: %w", err)
	}

	// 3. Provider setup
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	
	// Set global objects
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return &TelemetryEngine{tp: tp}, nil
}

// Shutdown gracefully flushes and stops the tracing pipeline.
func (e *TelemetryEngine) Shutdown(ctx context.Context) error {
	slog.Info("telemetry: shutting down tracing mesh")
	return e.tp.Shutdown(ctx)
}
