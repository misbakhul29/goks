package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

var registry = make(map[string]any)

// Register exposes a Go function to be called from the WASM client.
// The function must take exactly one argument and return (result, error).
func Register(name string, fn any) {
	registry[name] = fn
}

// Handler returns an HTTP handler that processes RPC requests.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := r.URL.Query().Get("method")
		fn, ok := registry[name]
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
