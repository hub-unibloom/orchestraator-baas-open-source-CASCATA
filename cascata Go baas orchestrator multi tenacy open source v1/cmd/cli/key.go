package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// handleKey routes commands targeting 'cascata key <subcommand>'
func handleKey(cfg CommandConfig, args []string) {
	if len(args) == 0 {
		Fatal(cfg, "MISSING_COMMAND", "Subcommand missing for key", "Valid options: rotate")
	}

	sub := args[0]
	switch sub {
	case "rotate":
		keyRotate(cfg, args[1:])
	default:
		Fatal(cfg, "UNKNOWN_COMMAND", "Unknown key command", sub)
	}
}

// keyRotate invalidates and rolls over core cryptographic boundaries (JWT, Anon, Service).
// CRITICAL: Force interactive password entry. It cannot be automated silently via scripts.
func keyRotate(cfg CommandConfig, args []string) {
	fs := flag.NewFlagSet("key-rotate", flag.ExitOnError)
	slug := fs.String("slug", "", "Project slug")
	keyType := fs.String("type", "", "Key to rotate (jwt|anon|service)")
	fs.Parse(args)

	if *slug == "" || *keyType == "" {
		Fatal(cfg, "MISSING_FLAGS", "Must provide --slug and --type", "")
	}

	validKeys := map[string]bool{"jwt": true, "anon": true, "service": true}
	if !validKeys[*keyType] {
		Fatal(cfg, "INVALID_KEY_TYPE", "Type must be one of: jwt|anon|service", *keyType)
	}

	if cfg.DryRun {
		impact := map[string]interface{}{
			"action": fmt.Sprintf("rotate_%s", *keyType),
			"target": *slug,
			"cascading_impact": []string{
				fmt.Sprintf("Invalidates all actively issued %ss", strings.ToUpper(*keyType)),
				"Drains all active WebSockets tied to this signature",
				"Forces logout on all UI sessions",
			},
			"dry_run": true,
		}
		PrintSuccess(cfg, impact, fmt.Sprintf("[DRY-RUN] Will instantly rotate %s keys for %s.", strings.ToUpper(*keyType), *slug))
		return
	}

	// SECURITY: Ensure human presence. AI or cron jobs cannot silently execute this phase.
	RequireInteractivity(cfg, "Cryptographic boundary resets require human authorization.")

	// Prompt for Master Password securely without echoing keystrokes
	fmt.Printf("\n[SECURITY] Rotating %s key for project %q.\n", strings.ToUpper(*keyType), *slug)
	fmt.Print("Enter Master Or Admin Password: ")

	// Note: terminal package relies on int file descriptor
	fd := int(os.Stdin.Fd())
	passwordBytes, err := term.ReadPassword(fd)
	fmt.Println() // emit newline after entry

	if err != nil {
		Fatal(cfg, "PROMPT_ERROR", "Failed to securely read password", err.Error())
	}

	password := strings.TrimSpace(string(passwordBytes))
	if len(password) < 8 {
		Fatal(cfg, "AUTH_FAILED", "Invalid password complexity or missing", "")
	}

	// 1. Authenticate internally with parsed password
	// authContext, err := verifyAdminPassword(slug, password)

	// 2. Trigger RPC/REST endpoint forcing generation of new keys via Vault Transit
	// newKeyStr := rotateVaultKeyRPC(ctx, authContext)

	// For demonstration in CLI binary mock:
	newKeyStr := fmt.Sprintf("sk_v1_%s_rolled_over", *keyType)

	result := map[string]interface{}{
		"action": "rotated",
		"type":   *keyType,
		"new_key": newKeyStr,
		"revoked_count": 8734, // Mock
	}

	PrintSuccess(cfg, result, fmt.Sprintf("\nSUCCESS: The %s root key was rotated. 8,734 active sessions invalidated.", strings.ToUpper(*keyType)))
}

// Support function in a non-tty environment that attempts to bind.
func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}
