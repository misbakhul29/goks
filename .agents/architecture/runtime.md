# Architecture Note: Runtime (`runtime/server`, `runtime/client`)

`runtime/server` wires together a compiled GoKS application on the server
side: middleware stack, router, and rendering pipeline. `runtime/client` is
the WASM entrypoint that boots hydration for interactive islands in the
browser.

Key invariants:
- A page with no interactive components must produce 0 KB of WASM payload.
- Standard middleware pipeline ordering:
  1. `router.RequestID()`: generates unique UUID / extracts `X-Request-Id` and injects into `context.Context` and response headers.
  2. `router.Logger()`: records method, path, response status, duration, and associated request ID.
  3. `router.Recover()`: catches downstream panics, logs stack, and returns HTTP 500 without crashing the process.
  4. Custom user middlewares (e.g. `CORS`, `Secure`, `Compress`, `MaxBytes`, `Timeout`).
- Graceful shutdown lifecycle:
  - Triggered via `DevServer.Shutdown(ctx)`.
  - Drains active in-flight HTTP connections via standard `http.Server.Shutdown(ctx)`.
  - Disconnects all connected `pkg/ws.Hub` clients cleanly by sending WebSocket close frames (`CloseGoingAway`), closing send channels, and invoking disconnect callbacks.

Before changing this package: check `docs/adr/` for any ADR touching runtime
lifecycle, and update `ARCHITECTURE.md`'s "Request Lifecycle" section if the
flow changes.
