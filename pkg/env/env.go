// Package env provides a unified way to read environment variables
// on both the server (os.Getenv) and the client browser (window.__GOKS_ENV).
package env

// Get retrieves the value of the environment variable named by the key.
// It returns the value, which will be empty if the variable is not present.
// Note: On the client-side (WASM), only variables prefixed with "PUBLIC_" are available.
func Get(key string) string {
	return getEnv(key)
}

// GetDefault retrieves the value of the environment variable named by the key,
// or returns fallback if the variable is not present or empty.
func GetDefault(key, fallback string) string {
	val := getEnv(key)
	if val == "" {
		return fallback
	}
	return val
}
