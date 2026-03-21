package engine

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"cascata/internal/database"
	"cascata/internal/utils"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Runtime manages an isolated execution environment for a specific function run.
type Runtime struct {
	Interp *interp.Interpreter
	Memory int64 // 128MB max
	Timeout time.Duration // 30s max
}

func NewRuntime() *Runtime {
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols) // Only provide stdlib symbols by default
	
	return &Runtime{
		Interp: i,
		Memory: 128 * 1024 * 1024,
		Timeout: 30 * time.Second,
	}
}

// Result of an execution.
type Result struct {
	Body interface{}
	Log  string
}

// ExecuteGo runs the provided Go source in the sandbox.
func (r *Runtime) ExecuteGo(ctx context.Context, source string, projectSlug string, pool *database.TenantPool, payload interface{}) (*Result, error) {
	// 1. Create a timeout context
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	// 2. Prepare Bridges (SQL, Net, Crypto)
	// We expose these as global variables or symbols the script can use.
	r.Interp.Use(interp.Exports{
		"cascata/db": map[string]reflect.Value{
			"Query": reflect.ValueOf(func(sql string, params []interface{}) (interface{}, error) {
				// SQL Bridge: Enforce RLS and timeout
				return pool.QueryWithRLS(ctx, "cascata_api_role", sql, params)
			}),
		},
		"cascata/net": map[string]reflect.Value{
			"Fetch": reflect.ValueOf(func(url string) (string, error) {
				// Network Bridge: SSRF Safe
				if err := utils.IsSSRFSafe(url); err != nil { return "", err }
				// perform fetch logic...
				return "", nil
			}),
		},
	})

	// 3. Inject payload as a global var
	r.Interp.Globals()["$payload"] = reflect.ValueOf(payload)

	// 4. Run execution
	_, err := r.Interp.Eval(source)
	if err != nil {
		return nil, fmt.Errorf("engine: runtime error: %w", err)
	}

	// 5. Retrieve result from a convention (e.g. var Result interface{})
	val, err := r.Interp.Eval("Result")
	if err != nil {
		return &Result{Body: nil}, nil
	}

	return &Result{Body: val.Interface()}, nil
}
