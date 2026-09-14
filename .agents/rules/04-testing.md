---
trigger: always_on
---

# Testing Rules

- Every new feature or bug fix includes a test. For bug fixes, write a
  failing test first that reproduces the bug, then fix it, and keep the test.
- Run `go test ./...` and `go test -race ./...` before considering a task
  done.
- Add benchmark tests (`go test -bench`) for performance-sensitive code,
  especially `pkg/orm`, `pkg/component`, `pkg/router`, and WASM hot paths.
- For `pkg/component` / GOX / WASM changes: verify the WASM build succeeds,
  check bundle size impact, and sanity-check hydration/interaction behavior.
- Prefer table-driven tests and avoid over-mocking; test real behavior
  against `pkg/orm`'s SQLite backend where feasible instead of mocking the
  database.
