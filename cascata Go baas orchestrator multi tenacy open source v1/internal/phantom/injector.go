package phantom

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

const PayloadVolume = "cascata_extension_payloads"

// Injector handles the physical extraction of extension files.
type Injector struct{}

// NewInjector initializes the phantom engine.
func NewInjector() *Injector {
	return &Injector{}
}

// ExtractFromImage runs a temporary container to copy extension files into the shared volume.
func (i *Injector) ExtractFromImage(ctx context.Context, source Source) error {
	slog.Info("phantom: starting extraction", "image", source.Image, "provides", source.Provides)

	// Magic command from Node.js implementation:
	// 1. /usr/local/lib/postgresql/ -> .so
	// 2. /usr/local/share/postgresql/extension/ -> .control/.sql
	// 3. /usr/lib/ -> OS native libs
	extractCmd := fmt.Sprintf(`
		mkdir -p /cascata_extensions/lib /cascata_extensions/share /cascata_extensions/os_lib &&
		cp -rn /usr/local/lib/postgresql/*.so* /cascata_extensions/lib/ 2>/dev/null || true &&
		cp -rn /usr/local/share/postgresql/extension/* /cascata_extensions/share/ 2>/dev/null || true &&
		cp -n /usr/lib/*.so* /cascata_extensions/os_lib/ 2>/dev/null || true &&
		echo PHANTOM_INJECT_OK
	`)

	// Execute via host docker (socket mounted)
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"-v", PayloadVolume+":/cascata_extensions",
		"--entrypoint", "sh",
		source.Image,
		"-c", extractCmd,
	)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("phantom.Extract: command failure: %w: %s", err, string(output))
	}

	if !strings.Contains(string(output), "PHANTOM_INJECT_OK") {
		return fmt.Errorf("phantom.Extract: unexpected output: %s", string(output))
	}

	slog.Info("phantom: injection complete", "image", source.Image, "duration", time.Since(start))
	return nil
}
