# Changelog

All notable changes to GoKS are documented in this file in accordance with [Keep a Changelog](https://keepachangelog.com/) and [Semantic Versioning](https://semver.org/).

## [v1.0.0] - 2026-09-14

### Major Milestone: Production General Availability (GA)
GoKS reaches `v1.0.0` with full API freeze, zero-npm runtime, single standalone-binary compilation, WebAssembly islands, streaming SSR, built-in ORM with atomic transactions, and comprehensive security hardening.

---

## [v0.23.0] - 2026-09-14
### M7: API Stability, Ecosystem & Release Candidate
- **API Freeze**: Frozen stable public APIs across `pkg/router`, `pkg/component`, `pkg/action`, `pkg/auth`, `pkg/orm`, `pkg/ws`, `pkg/store`, `pkg/metadata`, `pkg/image`, and `pkg/rbac`.
- **Compile Contract Verification**: Compile-time contract tests in `tests/compatibility/api_compat_test.go`.
- **Migration Guide**: Published `BREAKING_CHANGES.md` documenting the path from `v0.15.x` to `v1.0.0`.
- **Ecosystem**: Published `CONTRIBUTING.md` and `docs/BENCHMARKS.md`.

---

## [v0.22.0] - 2026-09-14
### M6: Production Hardening & Operability
- **Security Fuzz Testing**: Added native fuzz test suites:
  - `internal/compiler/gox_fuzz_test.go` (`FuzzTranspile`)
  - `pkg/router/router_fuzz_test.go` (`FuzzRouterMatch`)
  - `pkg/orm/orm_fuzz_test.go` (`FuzzSQLIdentifierValidation`)
  - `pkg/action/action_fuzz_test.go` (`FuzzActionOriginValidation`)
- **Operability & Hardening**: Created `docs/OPERABILITY.md` covering body limits, execution deadlines, secret redaction, atomic migrations, and rollback procedures.

---

## [v0.21.0] - 2026-09-14
### M5: CLI, Scaffolding & Developer Experience
- **`goks new` Scaffolding**: Added `--module` flag, app name validation, path traversal prevention, and directory overwrite protection (`internal/cli/new_test.go`).
- **Build Checks**: Added input and compiler validation for `goks build` (`internal/cli/build_test.go`).
- **CLI Reference & Deployment**: Published `docs/reference/CLI_REFERENCE.md` and `docs/DEPLOYMENT.md`.

---

## [v0.20.0] - 2026-09-14
### M4: GOX, Component Model, SSR & Universal Store
- **GOX Diagnostics**: Added `CompileError` with file, line, column, and visual source caret mapping in `internal/compiler`.
- **JSX Fragment Support**: Shorthand `<>...</>` and `<Fragment>...</Fragment>` syntax.
- **Deterministic Compilation**: Deterministic sorting of HTML attributes and component props.
- **Hook Invariant & SSR Fallback**: Enforced hook call-order invariants and provided SSR initial state fallback without panics.
- **SSR Isolation**: Prevented fiber context leakage across concurrent server requests.
- **Universal Store**: Made `pkg/store` universal with thread-safe unique ID subscription mapping.
- **Zero-WASM Static Verification**: Verified 0 KB WASM emission on purely static pages.

---

## [v0.19.0] - 2026-09-14
### M3: ORM Safety, Transactions & Authentication
- **ORM Injection Protection**: Added identifier validation to table and column names in `pkg/orm/crud.go`.
- **Atomic Transactions**: Introduced `db.Transaction(ctx, fn)` with automatic rollback on error or panic.
- **Soft-Delete Restoration**: Added `orm.Restore` and `query.WithTrashed()`.
- **Auth Fixation Defense**: Invalidated prior sessions upon login and added `Manager.Close()` cleanup.

---

## [v0.18.0] - 2026-09-14
### M2: Routing Semantics & Error Model
- **Route Precedence**: Explicit precedence scoring (`static 100 > dynamic 10 > wildcard 1`).
- **HTTP 405 Method Not Allowed**: Route discovery with `Allow` response header.
- **URL Parameter Decoding**: Safe `url.PathUnescape` decoding for URL parameters.
- **Standardized Error Model**: Structured `APIError` and `ErrorResponse` with Request ID tracking.

---

## [v0.17.0] - 2026-09-14
### M1: HTTP Runtime & Operability Baseline
- **Middleware Standardization**: Ordered middleware chain (`Logger`, `Recover`, `RequestID`, `Secure`, `MaxBytes`, `Timeout`).
- **Graceful Shutdown**: Added `server.Shutdown(ctx)` with clean connection draining.
- **Health Probes**: Built-in `/_goks/healthz` and `/_goks/ready` endpoints.
- **Concurrency & Race Baseline**: Zero-race verification under concurrent load.

---

## [v0.16.0] - 2026-09-14
### M0: Foundation, API Inventory & Security Baseline
- **API Inventory**: Created `docs/api/API_INVENTORY.md` classifying stable vs experimental APIs.
- **Security SLA**: Defined 48h vulnerability response SLA in `SECURITY.md`.
- **Unified Versioning**: Standardized on `internal/version` as single source of truth.
