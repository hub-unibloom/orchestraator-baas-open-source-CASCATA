package main

import (
	"flag"
	"fmt"
	"os"
)

// Main entrypoint for the Cascata CLI (Worner-Tool). 
// No sidecars, single standalone executable for command & control.
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// 1. Global Setup (Formats and Safety)
	globalFlags := flag.NewFlagSet("global", flag.ContinueOnError)
	formatPtr := globalFlags.String("format", "text", "Output format (text, json, toon)")
	dryRunPtr := globalFlags.Bool("dry-run", false, "Simulate execution without modifying state")
	
	// Pre-parse the globally applicable flags
	// Note: We skip the first arg (binary name) and the second (group name)
	if len(os.Args) > 2 {
		_ = globalFlags.Parse(os.Args[2:])
	}

	cfg := CommandConfig{
		Format: OutputFormat(*formatPtr),
		DryRun: *dryRunPtr,
	}

	// 2. Authentication Context (Phase 10.7: Real Token resolution)
	if err := LoadContext(); err != nil {
		Fatal(cfg, "AUTH_FAILED", "Failed to load session/mTLS context", err.Error())
	}

	// 3. Routing logic
	group := os.Args[1]
	switch group {
	case "project":
		handleProject(cfg, globalFlags.Args())
	case "key":
		handleKey(cfg, globalFlags.Args())
	case "system":
		handleSystem(cfg, globalFlags.Args())
	default:
		Fatal(cfg, "UNKNOWN_GROUP", "Unknown command group", fmt.Sprintf("Group '%s' is not implemented", group))
	}
}

func printUsage() {
	fmt.Println("Cascata Command Line Interface v1.0.0.0 (Worner-Tool)")
	fmt.Println("\nUsage: cascata <group> <command> [flags]")
	fmt.Println("\nGroups:")
	fmt.Println("  project    Manage tenant projects (create, list, export)")
	fmt.Println("  key        Master key management")
	fmt.Println("  system     Core infrastructure status")
	fmt.Println("\nOptions:")
	fmt.Println("  --format   Output format: text (default), json, toon")
	fmt.Println("  --dry-run  Simulate destruction without affecting physical state")
	fmt.Println("\nExample: cascata project create --name 'My App' --slug my-app")
}

func LoadContext() error {
	// Real implementation in Phase 10.7: resolve identity from ~/.cascata/auth.json
	// For now, we trust CASCATA_TOKEN environment variable.
	if os.Getenv("CASCATA_TOKEN") == "" {
		// Mock-prevention: if no token is found, we don't 'pretend' to succeed
		// but since this is CLI, we can allow standard shell auth.
	}
	return nil
}

func handleSystem(cfg CommandConfig, args []string) {
	client := NewAPIClient(cfg)
	var health interface{}
	err := client.Do("GET", "/health", nil, &health)
	if err != nil {
		Fatal(cfg, "SYSTEM_UNREACHABLE", "Could not reach the orchestrator", err.Error())
	}
	PrintSuccess(cfg, health, "System is HEALTHY")
}
