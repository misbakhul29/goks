package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]any)
)

// Register exposes a Go function to be called from the WASM client.
// The function must take exactly one argument and return (result, error).
func Register(name string, fn any) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = fn
}

// Handler returns an HTTP handler that processes RPC requests.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				http.Error(w, "internal RPC error", http.StatusInternalServerError)
			}
		}()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Prevent Cross-Site Request Forgery (CSRF) by requiring same-origin
		origin := r.Header.Get("Origin")
		if origin != "" && !strings.Contains(origin, r.Host) {
			http.Error(w, "cross-origin RPC not allowed", http.StatusForbidden)
			return
		}

		// Limit RPC payload to 1MB to prevent memory exhaustion DoS
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		name := r.URL.Query().Get("method")
		registryMu.RLock()
		fn, ok := registry[name]
		registryMu.RUnlock()

		if !ok {
			http.Error(w, fmt.Sprintf("rpc method not found: %s", name), http.StatusNotFound)
			return
		}

		fnVal := reflect.ValueOf(fn)
		fnType := fnVal.Type()

		if fnType.NumIn() != 1 {
			http.Error(w, "rpc function must have exactly 1 argument", http.StatusBadRequest)
			return
		}

		argType := fnType.In(0)
		argPtr := reflect.New(argType)
		if err := json.NewDecoder(r.Body).Decode(argPtr.Interface()); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		out := fnVal.Call([]reflect.Value{argPtr.Elem()})

		if len(out) == 2 && !out[1].IsNil() {
			err := out[1].Interface().(error)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(out[0].Interface()); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}
