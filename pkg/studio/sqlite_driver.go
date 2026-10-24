//go:build !js && !wasm

package studio

// Register the pure-Go SQLite driver for native (server) builds so that
// studio inspect features work with zero configuration.
//
// The driver is excluded from js/wasm builds: modernc.org/libc has no
// js/wasm support, and SQLite is a server-side concern only.
import _ "modernc.org/sqlite"
