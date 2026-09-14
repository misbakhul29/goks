# Architecture Note: Runtime (`runtime/server`, `runtime/client`)

`runtime/server` wires together a compiled GoKS application on the server
side: middleware stack, router, and rendering pipeline. `runtime/client` is
the WASM entrypoint that boots hydration for interactive islands in the
browser.

Key invariants:
- A page with no interactive components must produce 0 KB of WASM payload.
- `runtime/server` must apply the middleware stack (Logger, CORS, Secure,
  RequestID, Compress, MaxBytes, Timeout) consistently regardless of which
  route matched.
- Graceful shutdown must drain in-flight requests and close `pkg/ws`
  connections cleanly.

Before changing this package: check `docs/adr/` for any ADR touching runtime
lifecycle, and update `ARCHITECTURE.md`'s "Request Lifecycle" section if the
flow changes.
