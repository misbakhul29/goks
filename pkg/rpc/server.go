package rpc

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sort"
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

// RegisteredMethods returns a sorted list of all registered RPC method names.
func RegisteredMethods() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	methods := make([]string, 0, len(registry))
	for m := range registry {
		methods = append(methods, m)
	}
	sort.Strings(methods)
	return methods
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

		// CSRF protection: validate Sec-Fetch-Site, Origin, and Referer.
		if secSite := r.Header.Get("Sec-Fetch-Site"); secSite == "cross-site" {
			http.Error(w, "cross-origin RPC not allowed", http.StatusForbidden)
			return
		}

		origin := r.Header.Get("Origin")
		if origin != "" {
			if !sameOrigin(r, origin) {
				http.Error(w, "cross-origin RPC not allowed", http.StatusForbidden)
				return
			}
		} else if referer := r.Header.Get("Referer"); referer != "" {
			if !sameOriginReferer(r, referer) {
				http.Error(w, "cross-origin RPC not allowed", http.StatusForbidden)
				return
			}
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
			log.Printf("[GoKS/rpc] error executing %s: %v", name, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(out[0].Interface()); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

func sameOrigin(r *http.Request, rawOrigin string) bool {
	origin, err := url.Parse(rawOrigin)
	if err != nil || origin.Scheme == "" || origin.Host == "" || origin.User != nil ||
		origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return false
	}

	requestScheme := "http"
	if r.TLS != nil {
		requestScheme = "https"
	}
	if r.URL != nil && r.URL.Scheme != "" {
		requestScheme = strings.ToLower(r.URL.Scheme)
	}
	if !strings.EqualFold(origin.Scheme, requestScheme) {
		return false
	}

	requestHost := r.Host
	if requestHost == "" && r.URL != nil {
		requestHost = r.URL.Host
	}
	requestURL, err := url.Parse(requestScheme + "://" + requestHost)
	if err != nil {
		return false
	}

	return strings.EqualFold(strings.TrimSuffix(origin.Hostname(), "."), strings.TrimSuffix(requestURL.Hostname(), ".")) &&
		effectivePort(origin) == effectivePort(requestURL)
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	return "80"
}

func sameOriginReferer(r *http.Request, rawReferer string) bool {
	if strings.HasPrefix(rawReferer, "/") && !strings.HasPrefix(rawReferer, "//") {
		return true
	}
	refURL, err := url.Parse(rawReferer)
	if err != nil || refURL.Scheme == "" || refURL.Host == "" {
		return false
	}
	originFromRef := refURL.Scheme + "://" + refURL.Host
	return sameOrigin(r, originFromRef)
}
