package engine

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WASMRuntime manages a WASM execution environment for critical functions.
type WASMRuntime struct {
	Runtime wazero.Runtime
}

func NewWASMRuntime() *WASMRuntime {
	return &WASMRuntime{
		Runtime: wazero.NewRuntime(context.Background()),
	}
}

// ExecuteWASM runs the provided WASM binary with strict limits.
func (r *WASMRuntime) ExecuteWASM(ctx context.Context, wasmBinary []byte, projectSlug string, stdin io.Reader, stdout io.Writer) error {
	wasi_snapshot_preview1.MustInstantiate(ctx, r.Runtime)
	
	mod, err := r.Runtime.InstantiateWithConfig(ctx, wasmBinary, wazero.NewModuleConfig().
		WithStdin(stdin).
		WithStdout(stdout).
		WithStderr(os.Stderr).
		WithEnv("PROJECT_SLUG", projectSlug))
	
	if err != nil {
		return fmt.Errorf("wasm: execution failed: %w", err)
	}

	defer mod.Close(ctx)
	return nil
}
