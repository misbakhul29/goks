//go:build !js && !wasm

package orm

// Register the pure-Go SQLite driver for native (server) builds so that
// orm.OpenSQLite and the `goks db` CLI work with zero configuration.
//
// The driver is excluded from js/wasm builds: modernc.org/libc has no
// js/wasm support, and SQLite is a server-side concern only.
import _ "modernc.org/sqlite"
