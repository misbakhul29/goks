# Roadmap Detail: P1 — Runtime

## Goal
Harden the HTTP/runtime core so it is dependable under real traffic.

## Items
- Verify and document middleware ordering and behavior (`Logger`, `CORS`,
  `Secure`, `RequestID`, `Compress`, `MaxBytes`, `Timeout`).
- Confirm graceful shutdown drains in-flight requests and closes `pkg/ws`
  connections without dropping messages mid-write where avoidable.
- Review `pkg/ws` hub for goroutine leaks and race conditions
  (`go test -race`).
- Baseline structured logging across `runtime/server` request handling.

## Acceptance Criteria
- Race detector clean for `pkg/ws` and `runtime/server` under concurrent load
  test.
- Documented middleware order in `.agents/architecture/runtime.md`.
