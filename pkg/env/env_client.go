//go:build js && wasm

package env

import "syscall/js"

// getEnv retrieves the environment variable from the global JavaScript object
// injected by the GoKS server runtime.
func getEnv(key string) string {
	global := js.Global().Get("__GOKS_ENV")
	if global.IsUndefined() || global.IsNull() {
		return ""
	}

	val := global.Get(key)
	if val.IsUndefined() || val.IsNull() {
		return ""
	}

	return val.String()
}
