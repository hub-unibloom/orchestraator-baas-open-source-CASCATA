package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"cascata/internal/domain"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	AutomationStreamKey = "cascata:automation:events"
	ConsumerGroupName   = "automation-worker-group"
)

// EventQueue manages the asynchronous delivery of database changes to the workflow engine.
// It uses Dragonfly Streams for durability and scalability under high pressure (Phase 10).
type EventQueue struct {
	dfly *redis.Client
}

func NewEventQueue(dfly *redis.Client) *EventQueue {
	return &EventQueue{dfly: dfly}
}

// Publish pushes a new database event into the global automation stream.
func (q *EventQueue) Publish(ctx context.Context, event *domain.AutomationEvent) error {
	// 1. Context Propagation (Phase 17 Sinergy)
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		event.TraceID = span.SpanContext().TraceID().String()
		event.SpanID = span.SpanContext().SpanID().String()
	}

	b, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("queue: serialization failed: %w", err)
	}

	// 2. XADD to the stream (Auto-Trim to 10k messages for performance)
	err = q.dfly.XAdd(ctx, &redis.XAddArgs{
		Stream: AutomationStreamKey,
		MaxLen: 10000,
		Approx: true,
		Values: map[string]interface{}{"payload": string(b)},
	}).Err()

	if err != nil {
		slog.Error("queue: XADD failure", "error", err)
		return err
	}

	slog.Debug("queue: event published to stream", "table", event.Table, "slug", event.ProjectSlug)
	return nil
}

// ListenAndServe consumer group and processes events using the provided engine.
// It implements the Worker pattern for Phase 10.
func (q *EventQueue) ListenAndServe(ctx context.Context, engine *WorkflowEngine, workerID string) {
	slog.Info("queue: starting automation worker", "id", workerID)

	// 2. Ensure Consumer Group exists
	_ = q.dfly.XGroupCreateMkStream(ctx, AutomationStreamKey, ConsumerGroupName, "0").Err()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 3. XREADGROUP from the stream
			streams, err := q.dfly.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    ConsumerGroupName,
				Consumer: workerID,
				Streams:  []string{AutomationStreamKey, ">"},
				Count:    10,
				Block:    5 * time.Second,
			}).Result()

			if err != nil && err != redis.Nil {
				slog.Error("queue: read failure", "error", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// 4. Process Batch
			for _, stream := range streams {
				for _, msg := range stream.Messages {
					var event domain.AutomationEvent
					payload, _ := msg.Values["payload"].(string)
					
					if err := json.Unmarshal([]byte(payload), &event); err == nil {
						// 4.1. Extract Link (Context Sinergy - Phase 17)
						var linkedCtx context.Context
						if event.TraceID != "" {
							tid, _ := trace.TraceIDFromHex(event.TraceID)
							sid, _ := trace.SpanIDFromHex(event.SpanID)
							
							spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
								TraceID: tid,
								SpanID:  sid,
								Remote:  true,
							})
							linkedCtx = trace.ContextWithSpanContext(ctx, spanCtx)
						} else {
							linkedCtx = ctx
						}

						// 4.2. Start Consumer Span (Tracing Sinergy)
						tr := otel.Tracer("automation-worker")
						cCtx, span := tr.Start(linkedCtx, fmt.Sprintf("consume %s.%s", event.ProjectSlug, event.Table),
							trace.WithAttributes(
								attribute.String("messaging.system", "dragonfly"),
								attribute.String("messaging.destination", AutomationStreamKey),
								attribute.String("cascata.project", event.ProjectSlug),
							))
						
						slog.Info("automation: event consumed", "id", msg.ID, "slug", event.ProjectSlug, "trace_id", event.TraceID)
						
						// 4.3. TODO: engine.ExecuteWorkflow(cCtx, ...)
						_ = cCtx
						
						span.End()
					}

					// 5. ACK the message
					q.dfly.XAck(ctx, AutomationStreamKey, ConsumerGroupName, msg.ID)
				}
			}
		}
	}
}
