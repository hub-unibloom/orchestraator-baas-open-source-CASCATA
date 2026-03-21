package main

import (
	"flag"
	"fmt"
	"strings"

	"golang.org/x/net/idna"
)

// handleTenant routes commands targeting 'cascata tenant <subcommand>'
func handleTenant(cfg CommandConfig, args []string) {
	if len(args) == 0 {
		Fatal(cfg, "MISSING_COMMAND", "Subcommand missing for tenant", "Valid options: list, create, delete, info, suspend, restore")
	}

	sub := args[0]
	switch sub {
	// case "create":
	// 	tenantCreate(cfg, args[1:])
	case "delete":
		tenantDelete(cfg, args[1:])
	case "info":
		tenantInfo(cfg, args[1:])
	default:
		Fatal(cfg, "UNKNOWN_COMMAND", "Unknown tenant command", sub)
	}
}

// tenantInfo resolves a tenant by Slug or Custom Domain. 
// Uses Punycode conversion if a non-ASCII domain (e.g., maçã.com) is provided.
func tenantInfo(cfg CommandConfig, args []string) {
	fs := flag.NewFlagSet("tenant-info", flag.ExitOnError)
	slug := fs.String("slug", "", "Project slug")
	domain := fs.String("domain", "", "Custom domain (supports IDNA)")
	fs.Parse(args)

	if *slug == "" && *domain == "" {
		Fatal(cfg, "MISSING_FLAG", "Must provide --slug or --domain", "")
	}

	searchKey := *slug
	if *domain != "" {
		puny, err := NormalizeDomain(*domain)
		if err != nil {
			Fatal(cfg, "INVALID_DOMAIN", "Failed to normalize unicode domain", err.Error())
		}
		searchKey = puny
	}

	// 1. Action: Trigger the internal core's GetProject (Mocked RPC)
	// getProjectRPC(ctx, searchKey)

	// In a real environment, the CLI leverages Go net/rpc, gRPC, or an internal HTTP admin edge
	// to fetch the `internal/service/orchestrator` without pulling the massive database dependencies into the tiny binary.
	
	// Format to original for UI Display
	displayDomain := searchKey
	if *domain != "" {
		unicodeDomain, _ := idna.ToUnicode(searchKey)
		displayDomain = unicodeDomain
	}

	response := map[string]interface{}{
		"id": "e4b9-823a-c11f",
		"slug": searchKey,
		"domain": displayDomain,
		"status": "active",
		"memory_mb": 512,
	}

	PrintSuccess(cfg, response, fmt.Sprintf("Tenant %s (%s) is ACTIVE", searchKey, displayDomain))
}

// tenantDelete simulates a destructive action enforcing '--dry-run'.
func tenantDelete(cfg CommandConfig, args []string) {
	fs := flag.NewFlagSet("tenant-delete", flag.ExitOnError)
	slug := fs.String("slug", "", "Project slug to destroy")
	fs.Parse(args)

	if *slug == "" { Fatal(cfg, "MISSING_FLAG", "Must provide --slug", "") }

	if cfg.DryRun {
		// Agent Simulation Payload. Tells the AI what _would_ happen without blowing up production.
		impact := map[string]interface{}{
			"action": "delete_tenant",
			"target": *slug,
			"cascading_impact": []string{
				"Drops 43 public tables",
				"Deletes 12 edge functions",
				"Wipes 1.2TB of S3 Storage via VFS",
				"Revokes 102 active JWT tokens",
			},
			"dry_run": true,
		}
		PrintSuccess(cfg, impact, fmt.Sprintf("[DRY-RUN] Would delete tenant %s and 1.2TB of data.", *slug))
		return
	}

	// Active execution...
	// Trigger internal backend deletion.
	
	response := map[string]interface{}{
		"action": "delete_tenant",
		"target": *slug,
		"deleted": true,
	}
	PrintSuccess(cfg, response, fmt.Sprintf("Tenant %s completely wiped from existence.", *slug))
}
