package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Standard Cascata Error Codes (AI semantic parsers)
const (
	ErrCodeAuthFailed     = "AUTH_FAILED"
	ErrCodeRLSViolation   = "RLS_VIOLATION"
	ErrCodeQuotaExceeded  = "QUOTA_EXCEEDED"
	ErrCodeTenantNotFound = "TENANT_NOT_FOUND"
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeInternal       = "INTERNAL_ERROR"
)

// APIError forms the standard contract for every failed request.
// It abolishes free-text error logging favoring structured payload evaluation.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error forces APIError to implement the native `error` interface.
func (e *APIError) Error() string {
	return e.Message
}

// WriteError ensures the HTTP response strictly follows the Cascata Protocol,
// preventing 200 OK statuses on failed operations.
func WriteError(w http.ResponseWriter, r *http.Request, statusCode int, code, message, details string) {
	// Record in Telemetry Span if present
	span := trace.SpanFromContext(r.Context())
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("error.code", code),
			attribute.String("error.message", message),
		)
		span.RecordError(fmt.Errorf("%s: %s", code, message))
	}

	if statusCode < 400 {
		statusCode = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := APIError{
		Code:    code,
		Message: message,
		Details: details,
	}

	// Never panic during error rendering.
	json.NewEncoder(w).Encode(response)
}

// HandlePanic provides a defers-wrapper to catch rogue unhandled runtime errors 
// and convert them to the expected struct, avoiding raw stacktraces bleeding 
// out to the AI or end-users.
func HandlePanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rcv := recover(); rcv != nil {
				// Record Panic in OTel
				span := trace.SpanFromContext(r.Context())
				span.SetAttributes(attribute.Bool("error.panic", true))
				span.RecordError(fmt.Errorf("panic: %v", rcv))

				// Respond structurally
				WriteError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Critical runtime failure in orchestrator", "Check system logs for recovery details")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
