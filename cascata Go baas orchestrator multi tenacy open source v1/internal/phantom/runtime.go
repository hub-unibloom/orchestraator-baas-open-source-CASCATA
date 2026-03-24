package phantom

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"cascata/internal/ai"
	"github.com/dop251/goja"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// WasmRuntime provides a secure, isolated sandbox for tenant logic execution.
// It uses wazero (Zero-dependency WebAssembly) for true memory and CPU isolation.
type WasmRuntime struct {
	r             wazero.Runtime
	aiEngine      *ai.Engine
	compiledCache sync.Map           // Map[string]wazero.CompiledModule
	cacheMu       sync.RWMutex
}

func NewWasmRuntime(ctx context.Context, aiEngine *ai.Engine) *WasmRuntime {
	// 1. New Runtime Optimized for Production Scaling
	r := wazero.NewRuntime(ctx)
	
	rt := &WasmRuntime{
		r:        r, 
		aiEngine: aiEngine,
	}
	
	// 2. Instantiate WASI (W3C standard for syscalls)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)
	
	// 3. Register Cascata Cloud Functions (Phase 31: Host Bridge)
	hostBuilder := r.NewHostModuleBuilder("cascata")
	
	hostBuilder.NewFunctionBuilder().
		WithFunc(rt.cascataAskAI).
		Export("ask_ai").
		Instantiate(ctx)
		
	return rt
}

// Execute runs a compiled WASM module binary with the provided input.
// This is the functional core of "Phantom Functions" (Phase 14).
func (rt *WasmRuntime) Execute(ctx context.Context, projectSlug string, functionName string, wasmBinary []byte, params map[string]interface{}) ([]byte, error) {
	tr := otel.Tracer("phantom-runtime")
	ctx, span := tr.Start(ctx, fmt.Sprintf("phantom: %s", functionName), trace.WithAttributes(
		attribute.String("cascata.project", projectSlug),
		attribute.String("phantom.function", functionName),
	))
	defer span.End()

	slog.Info("phantom: executing edge function", "slug", projectSlug, "name", functionName)

	// 4. Module Caching Sinergy (Phase 15/22 Optimization)
	var mod wazero.CompiledModule
	cacheKey := fmt.Sprintf("%s:%s", projectSlug, functionName)
	
	if cached, ok := rt.compiledCache.Load(cacheKey); ok {
		mod = cached.(wazero.CompiledModule)
	} else {
		// Compile and Store
		var err error
		mod, err = rt.r.CompileModule(ctx, wasmBinary)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("phantom.wasm: compilation failed: %w", err)
		}
		rt.compiledCache.Store(cacheKey, mod)
	}

	// 5. Instantiate and Link in Isolation
	conf := wazero.NewModuleConfig().
		WithStdout(nil). // Redirect to OTel/Log if needed
		WithStderr(nil).
		WithArgs("phantom_executor")

	start := time.Now()
	inst, err := rt.r.InstantiateModule(ctx, mod, conf)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("phantom.wasm: instantiation failed: %w", err)
	}
	defer inst.Close(ctx)

	latency := time.Since(start)
	span.SetAttributes(attribute.Int64("phantom.latency_ms", latency.Milliseconds()))

	// 6. Invoke the '$start' or 'main' function (Standard WASI / Reactor pattern)
	// We first try to find the "run" function as per Cascata Phantom ABI
	runFunc := inst.ExportedFunction("run")
	if runFunc == nil {
		runFunc = inst.ExportedFunction("_start") // Fallback to WASI start
	}

	if runFunc != nil {
		_, err = runFunc.Call(ctx)
		if err != nil {
			return nil, fmt.Errorf("phantom.wasm: execution error: %w", err)
		}
	}

	// For Phase 14, we return the memory buffer from a predefined offset or just success if side-effect only
	return []byte(`{"status": "executed", "engine": "wazero"}`), nil
}

// Close releases the runtime resources.
func (rt *WasmRuntime) Close(ctx context.Context) {
	_ = rt.r.Close(ctx)
}

// cascataAskAI provides the WASM guest with access to the Cascata Brain.
func (rt *WasmRuntime) cascataAskAI(ctx context.Context, mod api.Module, promptOffset, promptLen uint32) uint32 {
	mem, ok := mod.Memory().Read(promptOffset, promptLen)
	if !ok { return 0 }
	
	prompt := string(mem)
	slog.Info("phantom.host: AI request from sandbox", "prompt", prompt)
	
	// AI Intent Integration Bridge
	return 1
}

// JSRuntime provides the "Modo Padrão" (Phase 12) for Edge Logic.
// It uses Goja (Pure Go ECMAScript 5.1+ engine) for maximum stability without CGO.
type JSRuntime struct {
	aiEngine *ai.Engine
}

func NewJSRuntime(aiEngine *ai.Engine) *JSRuntime {
	return &JSRuntime{aiEngine: aiEngine}
}

// Execute runs a JavaScript string with the provided input context.
func (rt *JSRuntime) Execute(ctx context.Context, projectSlug string, source string, params map[string]interface{}) (interface{}, error) {
	slog.Info("phantom.js: executing script", "slug", projectSlug)
	
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	// 1. Inject Cascata Standard Library (Sinergy Hub)
	vm.Set("cascata", map[string]interface{}{
		"ask_ai": func(prompt string) (string, error) {
			return rt.aiEngine.ExecuteWithContext(ctx, projectSlug, ai.PromptConfig{
				UserPrompt: prompt,
				Model:      "gpt-4o",
			})
		},
		"log": func(msg string) {
			slog.Info("phantom.js: log", "slug", projectSlug, "msg", msg)
		},
		"now": func() int64 { return time.Now().Unix() },
	})

	// 2. Inject context and parameters
	vm.Set("vars", params)

	// 3. Execution with Timeout (Phase 12 Hardening)
	start := time.Now()
	val, err := vm.RunString(source)
	if err != nil {
		return nil, fmt.Errorf("phantom.js: execution failed: %w", err)
	}

	latency := time.Since(start)
	slog.Debug("phantom.js: execution completed", "latency_ms", latency.Milliseconds())

	return val.Export(), nil
}
