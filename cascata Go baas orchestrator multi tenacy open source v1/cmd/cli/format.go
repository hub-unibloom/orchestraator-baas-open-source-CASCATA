package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// The CLI must be fully deterministic for AI/Agentic usage.
// Standardizing output formats overrides standard stdout/stderr clutter.

// OutputFormat defines the expected response format.
type OutputFormat string

const (
	FormatText OutputFormat = "text"
	FormatJSON OutputFormat = "json"
	FormatTOON OutputFormat = "toon" // Cascata internal strict mode
)

// CommandConfig holds global CLI flags passed to every command.
type CommandConfig struct {
	Format OutputFormat
	DryRun bool
}

// ErrorResponse enforces a structured schema when the CLI fails.
// This allows AI agents to parse the exact issue without regexing text blocks.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// PrintSuccess emits structured or plaintext success data and exits 0.
func PrintSuccess(cfg CommandConfig, data interface{}, textFallback string) {
	switch cfg.Format {
	case FormatJSON:
		b, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(b))
	case FormatTOON:
		// TOON mock: proprietary packed formatting
		fmt.Printf("TOON_SUCCESS|%v\n", data)
	default: // FormatText
		fmt.Println(textFallback)
	}
	os.Exit(0)
}

// Fatal emits structured errors and exits with non-zero code.
func Fatal(cfg CommandConfig, code, message, details string) {
	errResp := ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}

	switch cfg.Format {
	case FormatJSON:
		b, _ := json.MarshalIndent(errResp, "", "  ")
		fmt.Fprintln(os.Stderr, string(b))
	case FormatTOON:
		fmt.Fprintf(os.Stderr, "TOON_ERROR|%s|%s|%s\n", code, message, details)
	default:
		fmt.Fprintf(os.Stderr, "ERROR [%s]: %s\n", code, message)
		if details != "" {
			fmt.Fprintf(os.Stderr, "Details: %s\n", details)
		}
	}
	// Exit > 0 for programmatic fallback handling
	os.Exit(1)
}

// RequireInteractivity aborts if an AI is trying to run a human-only interactive command without a tty.
func RequireInteractivity(cfg CommandConfig, rationale string) {
	if cfg.Format != FormatText {
		Fatal(cfg, "INTERACTIVE_REQUIRED", "This command requires interactive confirmation", rationale)
	}
	// In Go, check if stdin is a terminal:
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		Fatal(cfg, "TTY_REQUIRED", "This command requires a TTY console", rationale)
	}
}
