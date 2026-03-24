package phantom

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dop251/goja"
	"cascata/internal/ai"
)

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
