package api

import (
	"encoding/json"
	"net/http"
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
func WriteError(w http.ResponseWriter, statusCode int, code, message, details string) {
	// Standard mapping if someone pushes 200 with an error. 
	// The HTTP spec dictates correct status codes, this reinforces it.
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
func HandlePanic(w http.ResponseWriter, r *http.Request) {
	if rcv := recover(); rcv != nil {
		// Log the stacktrace internally to APM/Dragonfly.
		// log.Printf("[PANIC] %s", rcv)
		
		// Respond structurally
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, "Critical runtime failure in orchestrator", "Please check system logs or report to admin")
	}
}
