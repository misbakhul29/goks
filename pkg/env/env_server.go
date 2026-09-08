//go:build !js || !wasm

package env

import "os"

// getEnv retrieves the environment variable from the OS.
func getEnv(key string) string {
	return os.Getenv(key)
}
