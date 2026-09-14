// Package version defines the GoKS framework version used by generated apps,
// tooling, and runtime diagnostics.
package version

import (
	"runtime/debug"
	"strings"
)

const (
	// ModulePath is the canonical Go module path for GoKS.
	ModulePath = "github.com/misbakhul29/goks"
	// Version is the source-tree fallback used for development builds.
	Version = "v0.18.0"
)

// Current returns the version embedded in build information when available.
// Local source builds and replace directives use the repository version as a
// stable fallback instead of leaking Go's "(devel)" marker to users.
func Current() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Version
	}

	if v := usableVersion(info.Main.Path, info.Main.Version); v != "" {
		return v
	}
	for _, dep := range info.Deps {
		if v := usableVersion(dep.Path, dep.Version); v != "" {
			return v
		}
	}
	return Version
}

func usableVersion(path, value string) string {
	if path != ModulePath || value == "" || value == "(devel)" {
		return ""
	}
	return strings.TrimSuffix(value, "+dirty")
}
