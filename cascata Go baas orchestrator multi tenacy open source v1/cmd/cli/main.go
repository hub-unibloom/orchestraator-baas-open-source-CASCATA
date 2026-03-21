package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/net/idna"
)

// Main entrypoint for the Cascata CLI binary. No sidecars, single standalone executable.
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: cascata <group> <command> [flags]")
		fmt.Println("Example: cascata tenant delete --slug xyz --dry-run --format json")
		os.Exit(1)
	}

	// 1. Global Setup (Formats and Safety)
	globalFlags := flag.NewFlagSet("global", flag.ContinueOnError)
	formatPtr := globalFlags.String("format", "text", "Output format (text, json, toon)")
	dryRunPtr := globalFlags.Bool("dry-run", false, "Simulate execution without modifying state")
	
	// Pre-parse the globally applicable flags
	globalFlags.Parse(os.Args[2:])

	cfg := CommandConfig{
		Format: OutputFormat(*formatPtr),
		DryRun: *dryRunPtr,
	}

	// 2. Authentication Context Mock. 
	// The binary expects cascata.member OTP or agent_id TLS logic here.
	if err := LoadContext(); err != nil {
		Fatal(cfg, "AUTH_FAILED", "Failed to load session/mTLS context", err.Error())
	}

	// 3. Routing
	group := os.Args[1]
	switch group {
	case "tenant":
		handleTenant(cfg, globalFlags.Args())
	case "key":
		handleKey(cfg, globalFlags.Args())
	// case "db":
	// case "fn":
	// case "automation":
	// case "audit":
	default:
		Fatal(cfg, "UNKNOWN_GROUP", "Unknown command group", fmt.Sprintf("Group '%s' is not implemented", group))
	}
}

func LoadContext() error {
	// Abstracted implementation checking `~/.cascata/auth.json` or mTLS certs 
	// ensuring the user is a valid operator capable of pushing to the internal services.
	return nil
}

// NormalizeDomain applies IDNA (Punycode) transformations so the DB 
// receives a compliant sequence (e.g. café.com -> xn--caf-dma.com). 
// The UI always unwraps it for readability.
func NormalizeDomain(raw string) (string, error) {
	p := idna.New(
		idna.BidiRule(),
		idna.ValidateLabels(true),
		idna.StrictDomainName(true),
	)
	
	puny, err := p.ToASCII(raw)
	if err != nil {
		return "", fmt.Errorf("domain normalization failed: %w", err)
	}
	return puny, nil
}
