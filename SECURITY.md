# Security Policy

## Scope

GoKS ships code that runs both server-side and, via WebAssembly, in the
browser. Every change to the following areas requires explicit security
review before merge:

- `pkg/action` — server actions, CSRF protection, form fallback.
- `pkg/auth` — JWT/session handling.
- `pkg/rbac` — authorization checks.
- `pkg/router` — request routing and matching.
- `pkg/ws` — WebSocket authentication and message handling.
- `internal/cli` — anything that writes files, runs subprocesses, or serves
  a dev/build server.

## Checklist for Security-Sensitive Changes

- Authentication: are credentials/tokens validated on every protected path?
- Authorization: does `pkg/rbac` get consulted before the action executes,
  not just before rendering a link/button?
- CSRF: do all state-changing server actions (both WASM and no-JS form
  fallback paths) carry and validate a CSRF token?
- XSS: is any user-controlled data rendered without escaping in `pkg/html`
  or generated `.gox` output?
- SSRF: does any server-side fetch (if introduced) validate/allow-list
  destinations?
- Path traversal: does any file-serving or CLI file-writing code
  canonicalize and validate paths (`internal/cli`, static asset serving)?
- Request limits: are body size limits (`MaxBytes`) and timeouts applied to
  new endpoints?
- Session security: are session tokens rotated/invalidated correctly on
  login/logout?
- Secrets: are secrets read via `pkg/env` and never hard-coded or logged?
- WebSocket: is the handshake authenticated, and are room/broadcast
  permissions checked per-message, not just at connect time?

## Reporting a Vulnerability

Do not open a public issue for a suspected vulnerability. Instead, contact
the maintainer privately (see repository owner contact on GitHub) with:

- A description of the issue and affected package/version.
- Steps to reproduce or a minimal proof of concept.
- Any suggested fix, if known.

## Non-Negotiable

Security protections must never be weakened for convenience, developer
experience, or to unblock a feature deadline. If a trade-off is genuinely
necessary, it must be documented in an ADR with the accepted risk explicit.
