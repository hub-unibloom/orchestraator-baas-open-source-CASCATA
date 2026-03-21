package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"slices"

	"cascata/internal/domain"
	"cascata/internal/domain/mcp"
)

// GovernanceService handles perimeter security and tool call limitations.
type GovernanceService struct {
	// TODO: Add Dragonfly cache lookup for Whitelists.
}

// NewGovernanceService initializes the perimeter security manager.
func NewGovernanceService() *GovernanceService {
	return &GovernanceService{}
}

// EnforcePerimeter checks if a project is authorized to communicate from a given source.
func (s *GovernanceService) EnforcePerimeter(ctx context.Context, p *domain.Project, clientIP string) error {
	// 1. IP Blocklist Check (Zero-Trust).
	if slices.Contains(p.Blocklist, clientIP) {
		slog.Warn("governance: request blocked by IP", "ip", clientIP, "slug", p.Slug)
		return fmt.Errorf("ip address %s blocked by project security rules", clientIP)
	}

	// 2. IP Whitelist (If configured, restrict to these).
	// Placeholder logic: p.Metadata["whitelist_ips"]
	return nil
}

// ValidateToolCall ensures an AI agent (MCP) isn't exceeding limits.
func (s *GovernanceService) ValidateToolCall(ctx context.Context, p *mcp.ToolCall) error {
	// 1. Maximum Rows enforcement.
	if p.RowsRequested > 1000 { // Default safety ceiling
		return fmt.Errorf("tool call exceeds maximum allowable rows (1000)")
	}

	// 2. URL Allowlist (For external MCP tools).
	parsed, err := url.Parse(p.TargetURL)
	if err != nil {
		return fmt.Errorf("invalid MCP target URL")
	}

	// Prevent private IP access (SSRF protection)
	ip := net.ParseIP(parsed.Hostname())
	if ip != nil && ip.IsPrivate() {
		return fmt.Errorf("governance: tool call blocked - private network access attempt")
	}

	return nil
}
