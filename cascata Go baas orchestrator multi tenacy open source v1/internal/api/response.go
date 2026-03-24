package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Response is the canonical envelope for every success signal in the Cascata system.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta carries orchestration-level telemetry details.
type Meta struct {
	LatencyMS int64  `json:"latency_ms,omitempty"`
	Timestamp int64  `json:"timestamp"`
	Project   string `json:"project,omitempty"`
	Mode      string `json:"mode,omitempty"`
}

// ErrorResponse defines the standard Failure Payload for the system (Phase 19).
type ErrorResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// Error forces ErrorResponse to implement the native `error` interface.
func (e *ErrorResponse) Error() string {
	return e.Message
}

// SendJSON delivers a success payload with standardized metadata.
func SendJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := Response{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Timestamp: time.Now().Unix(),
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// SendError delivers a failure signal with RLS-safe diagnostic codes (Phase 19).
// It is integrated with Telemetry to ensure every failure is traceable in the mesh.
func SendError(w http.ResponseWriter, r *http.Request, status int, code, message string, detail ...string) {
	// 1. Record in Telemetry Span
	span := trace.SpanFromContext(r.Context())
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("error.code", code),
			attribute.String("error.message", message),
		)
		span.SetStatus(trace.Status{Code: trace.SpanStatusCodeError, Description: message})
	}

	// 2. Prepare Payload
	var d string
	if len(detail) > 0 { d = detail[0] }

	errResp := ErrorResponse{
		Success: false,
		Code:    code,
		Message: message,
		Detail:  d,
	}

	// 3. Log internally
	slog.Warn("api.error_delivered", "status", status, "code", code, "message", message, "detail", d)

	// 4. Respond
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errResp)
}

// Common Error Codes as Constants to enforce consistency across the "Holy Grail".
const (
	ErrUnauthorized    = "UNAUTHORIZED_IDENTITY"
	ErrForbidden       = "INSUFFICIENT_PRIVILEGES_OR_RLS_BLOCK"
	ErrInvalidRequest  = "INVALID_PAYLOAD_OR_PARAMS"
	ErrRateLimit       = "QUOTA_EXCEEDED_RATE_LIMIT_TRIPPED"
	ErrGenesisFailure  = "GENESIS_TENANT_BIRTH_FAILED"
	ErrInternalError   = "HEART_OF_SYSTEM_FAILURE"
)
