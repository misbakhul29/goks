# GoKS Architecture and Production Readiness Audit

Audit baseline: v0.15.1
Audit scope: M0.1 from `docs/task/ROAD_TO_V1.md`
Audit mode: read-only source, test, architecture, ADR, and documentation review
Audit status: findings recorded; remediation starts with P0 security and correctness items

## 1. Executive summary

GoKS has a coherent framework direction and a working Go-only baseline. The repository
currently builds as native Go and as `js/wasm`; the existing unit tests, race tests,
vet, and package builds pass. The framework already contains substantial pieces for
routing, server actions, component rendering, SSR, streaming Suspense, ORM
migrations, authentication, OAuth, RBAC, WebSockets, CLI scaffolding, and Studio.

It is not yet safe to claim production readiness or v1.0.0 compatibility. The main
risk is not missing surface area; it is that several public or security-sensitive
paths have insufficient invariants and tests. The first implementation work should
harden request-origin validation, HTTP response behavior, ORM identifier/query
boundaries, session-cookie trust, runtime concurrency, and the build/release path.

The current test suite is a useful compile/regression suite, but it does not yet
prove production behavior. Package coverage from `go test -cover ./...` shows notable
gaps: router 27.6%, ORM 32.3%, auth 37.0%, WebSocket 9.9%, CLI 9.6%, and several
packages at 0% because they have no tests. Coverage is not a release criterion by
itself, but these numbers identify untested public behavior that must be covered by
contract and integration tests.

## 2. Evidence collected

The following commands were executed against the v0.15.1 baseline:

- `go list ./...` — passed.
- `go test -count=1 ./...` — passed.
- `go test -race -count=1 ./...` — passed. The shell printed a non-fatal job-control
  warning before the normal package results; all packages completed successfully.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `GOOS=js GOARCH=wasm go build ./...` — passed.
- `go test -cover ./...` — passed, with the coverage gaps recorded below.
- `go test -run='^$' -bench=. ./pkg/router ./pkg/component ./pkg/orm` — passed, but
  no benchmark cases were reported for those packages.
- `git diff --check` — passed.

The audit also reviewed `AGENTS.md`, `GOKS_CONSTITUTION.md`, `ARCHITECTURE.md`,
`ROADMAP.md`, `SECURITY.md`, the `.agents/rules` files, architecture notes, ADRs
0000–0007, and the implementation under `pkg/`, `runtime/`, and `internal/`.

## 2.1 Evidence references

Important findings are anchored to the current source lines below. Line numbers are
expected to move as fixes land; update this section with every related change.

| Finding | Evidence |
| --- | --- |
| Action origin substring check (fixed in this work) | `pkg/action/action.go:95-99` before fix; regression tests in `pkg/action/action_test.go:96-125` |
| Action redirect accepted request-controlled target (fixed in this work) | `pkg/action/action.go:156-164`; regression test in `pkg/action/action_test.go:127-145` |
| Action request limit | `pkg/action/action.go:102-103` |
| Response streaming and manual transfer framing | `pkg/router/context.go:113-127` |
| Redirect/write state contract | `pkg/router/context.go:130-137` |
| ORM global database and raw `Where` condition boundary | `pkg/orm/orm.go:35-60`, `pkg/orm/orm.go:95-99` |
| ORM SQL identifiers interpolated into CRUD queries | `pkg/orm/crud.go:71-100`, `pkg/orm/crud.go:192-200` |
| Migration transaction and applied-state behavior | `pkg/orm/migrate.go:29-103`, `pkg/orm/migrate.go:163-189` |
| Session trust of forwarded scheme and process-local GC | `pkg/auth/auth.go:67-78`, `pkg/auth/auth.go:164-177` |
| Global SSR mutex and global current path | `runtime/server/server.go:36`, `runtime/server/server.go:412-432` |
| Server shutdown does not define `ErrServerClosed` handling | `runtime/server/server.go:116-131` |
| Raw `innerHTML` SSR sink | `pkg/component/ssr.go:36-62` |
| Reconciler interface comparison and keyed diff | `pkg/component/reconciler.go:69-113`, `pkg/component/reconciler.go:127-147` |
| Unbounded per-boundary streaming goroutines and error output | `pkg/component/stream.go:64-129` |
| Build silently skips CSS failure and ignores tidy failure | `internal/cli/build.go:58-68`, `internal/cli/build.go:203-220` |

## 3. Verified architecture map

### 3.1 Request and server path

- `pkg/router` owns programmatic routing, route groups, path parameters, middleware,
  and the request `Context`.
- `runtime/server` assembles development/production HTTP behavior, filesystem pages,
  SSR, static files, live reload, metadata, and generated application routing.
- `pkg/action` exposes the progressive server-action endpoint with JSON and HTML-form
  modes.
- `pkg/ws` contains the WebSocket hub, clients, rooms, typed events, and middleware.
- `pkg/rpc` provides RPC-style server/client plumbing.

### 3.2 Rendering and browser path

- `pkg/component` contains the virtual node model, component lifecycle, hooks,
  reconciliation, SSR, islands, store, Suspense, and streaming SSR.
- `runtime/client` and the `js && wasm` component renderer apply patches and event
  listeners through `syscall/js`.
- `internal/compiler` translates GOX source into generated Go code and workspace
  files; it is internal, not a public `pkg/compiler` package.

### 3.3 Data and security path

- `pkg/orm` wraps `database/sql`, builds queries, maps model fields, and runs SQL
  migrations and seeders.
- `pkg/auth` provides session and JWT primitives; OAuth providers live under
  `pkg/auth/oauth`.
- `pkg/rbac` provides role/permission middleware.
- `pkg/env` and the CLI provide configuration/build inputs.

### 3.4 Tooling path

The CLI currently contains `new`, `dev`, `build`, `start`, `ui`, `generate`, `page`,
`db`, `studio`, `export`, `lsp`, and `version`. Generated applications use an entry
workspace under `.goks`; build commands invoke Go and optional TinyGo/Tailwind
binaries.

## 4. What is working

These are verified capabilities, not a claim that all production edge cases are
complete:

- Native package compilation and WASM target compilation succeed.
- The existing unit-test and race-test suites pass.
- Basic static and dynamic routing, groups, JSON responses, body binding, and
  middleware ordering have regression tests.
- Server actions have tests for JSON mode, HTML redirect mode, unknown actions, and
  a basic cross-origin rejection case.
- SSR escapes text and attribute values in the normal renderer path; tests cover
  generated document structure and a zero-WASM static page.
- Store mutation copies callbacks before notification, avoiding a lock-held callback
  deadlock in the normal path.
- Migration parsing, ordering, mock execution, and failure handling have tests.
- JWT signing/verification and minimum secret length have tests.
- OAuth code exists with PKCE/state-related behavior and has a package test suite.
- Version resolution is centralized in `internal/version`; the current release command
  reports v0.15.1.
- The repository has ADRs for the core product direction: Go-first, islands, GOX,
  filesystem routing, server actions, ORM, and state management.

## 5. Findings by severity

Severity meanings:

- P0 — exploitable security/correctness issue or release blocker; fix before adding
  broad feature surface.
- P1 — production reliability or public-contract risk; fix before release candidate.
- P2 — important completeness/DX/testing work; can follow the P0/P1 foundation.
- P3 — post-v1 enhancement or explicitly deferred scope.

### 5.1 P0: origin validation is not an exact-origin policy

Affected area confirmed: `pkg/action/action.go`. Other state-changing endpoints
must be checked for the same pattern before release.

The action handler accepted an origin when the request host appeared as a substring.
A host such as `evil-localhost:3000.example` must not be treated as
`localhost:3000`. Security checks must parse the origin URL and compare scheme,
host, and effective port exactly against an explicitly configured allowlist. Host
headers and forwarded headers must not become trusted security policy without a
proxy trust configuration.

Required remediation:

1. Introduce one small, tested exact-origin policy used by actions and other
   state-changing browser endpoints.
2. Define behavior when `Origin` is absent and when only `Referer` is present.
3. Add tests for substring-prefix attacks, scheme changes, default ports, IPv6,
   malformed origins, trusted reverse proxies, and multiple allowed origins.
4. Add a real CSRF token option for cookie-authenticated state-changing actions; an
   Origin check alone is not a complete CSRF strategy for every client/proxy path.

### 5.2 P0: unsafe HTML and DOM escape boundaries need an explicit policy

`pkg/component/ssr.go` intentionally renders `innerHTML` without escaping and the
WASM renderer sets `innerHTML` directly. That can be a valid escape hatch, but it is
currently a raw public prop with no visibly enforced trusted-content type or
sanitization contract. The same surface exists in streaming SSR.

Required remediation:

1. Document `innerHTML` as trusted-only and add an explicit safe-HTML type or an
   equivalent API boundary before v1.0.
2. Add negative tests proving normal text/attributes remain escaped and tests proving
   the trusted path is deliberate.
3. Audit event props and attribute names for dangerous DOM sinks and URL schemes.
4. Add a security review for streaming replacement templates and error messages.

### 5.3 P0: production builds can silently produce incomplete assets

`internal/cli/build.go` treats Tailwind CLI download/build failure as a skipped step
and still reports a successful build. A production artifact without the expected CSS
is not equivalent to a successful production build. `go mod tidy` errors are also
ignored in the build path.

Required remediation:

- Make required assets fail the build by default.
- Add an explicit `--allow-missing-css` or development-only mode if degraded output is
  needed.
- Return and print actionable errors from dependency preparation.
- Add a generated-app integration test that builds a real fixture and verifies server,
  WASM, CSS, `wasm_exec.js`, standalone assets, and exit status.

### 5.4 P1: HTTP response and runtime concurrency contracts are underspecified

The router `Context` tracks status and `written` state independently from the wrapped
writer. `Redirect` marks the response written before calling `http.Redirect`; repeated
writes and status changes need a documented contract. `Bind` closes the request body,
which may surprise middleware or handlers that need to inspect it later. `StreamHTML`
sets `Transfer-Encoding` manually even though Go's HTTP server controls transfer
framing.

The server runtime also contains global/current-path and SSR coordination state. The
SSR path uses a package-level mutex, which prevents some races but serializes unrelated
requests and makes request-scoped behavior difficult to reason about. Production
runtime behavior needs an explicit request lifecycle and graceful-shutdown contract.

Required remediation:

- Define one response-writer state machine and test status/body/header behavior.
- Preserve or explicitly replace request-body ownership semantics.
- Use `http.Flusher` without manually forcing transfer encoding.
- Remove request-specific global state; pass route/path/render state through request
  context or a server instance.
- Add concurrent `httptest` integration tests and shutdown tests.

### 5.5 P1: ORM SQL identifier and database support boundaries are incomplete

The ORM parameterizes values in some query paths and validates `OrderBy`, but schema
and migration APIs still construct SQL from table, column, index, and migration names.
Those inputs need strict identifier validation or dialect-aware quoting. A caller
passing user-controlled identifiers must never be able to inject SQL.

The module manifest does not include a database driver. The ORM contains dialect
support and tests using a mock driver, but actual SQLite/PostgreSQL/MySQL support and
transaction semantics are not proven by the repository tests. Documentation must not
promise a database until a supported driver/integration matrix exists, or the
framework must define an explicit driver-injection contract.

Required remediation:

- Centralize identifier validation/quoting for every schema/query path.
- Add integration tests against each supported database, preferably in a container or
  a documented external-service CI job.
- Define connection ownership, `Close`, pool settings, context cancellation, and
  transaction APIs.
- Add migration locking, dirty/partial migration state, checksum/version policy,
  rows-error checks, and rollback tests.
- Document the driver installation and supported dialect matrix accurately.

### 5.6 P1: session and proxy security need a deployment model

The session implementation must define how `Secure` is determined. Trusting
`X-Forwarded-Proto` from an arbitrary client is unsafe unless the application has an
explicit trusted-proxy configuration. Cookie attributes, rotation, expiry,
logout/revocation, session-store limits, and cross-instance operation need integration
tests.

Required remediation:

- Default to secure cookie behavior in production and use a trusted-proxy policy for
  forwarded scheme data.
- Set and test `SameSite`, `Path`, expiry, and deletion attributes consistently.
- Add session fixation, concurrent-login, expiry, tampering, and restart tests.
- Define a shared session-store interface only if a real multi-instance use case is
  supported; otherwise document the process-local limitation.
- Add JWT algorithm/key rotation and issuer/audience validation requirements.

### 5.7 P1: component reconciliation and browser resource ownership need hardening

The reconciler compares arbitrary prop values with `!=`; interface values containing
maps, slices, or functions can panic unless they are handled before comparison.
Keyed-list matching is present but needs duplicate-key, move, deletion, and nested
path tests. The WASM renderer stores event listeners globally and allocates IDs
without an explicit renderer lifecycle; multiple renderers and long-lived pages need
resource-isolation tests.

Required remediation:

- Define comparable prop types or implement safe equality semantics.
- Add keyed reconciliation property/invariant tests.
- Tie listener registries and JS function release to renderer ownership.
- Test repeated updates, unmount, replacement, nested components, and multiple roots
  in a browser-capable environment.
- Define hook order/count invariants and fail predictably when violated.

### 5.8 P1: streaming SSR needs bounded work and safe cancellation

`RenderToStream` starts one goroutine per Suspense boundary and waits for results. It
has per-boundary timeouts, but production policy still needs limits on boundary count,
request lifetime, output buffering, client-disconnect behavior, and error disclosure.
Replacement scripts contain server-generated HTML and error text, so the output policy
must be tested as an HTML/JS security boundary.

Required remediation:

- Bound concurrent async work per request and cancel all work on disconnect.
- Define ordering guarantees and behavior after the writer fails.
- Avoid returning raw internal resolver errors to clients; log a correlation ID and
  render a safe public error.
- Add streaming integration tests for slow, failed, canceled, nested, and many-boundary
  requests.

## 6. Testing and verification gaps

The passing suite currently proves compilation and selected unit behavior, not the
full product contract. The following must become release-gate tests:

- Router: method mismatch (405/Allow), HEAD/OPTIONS, duplicate routes, precedence,
  encoded paths, traversal-like paths, wildcard edge cases, concurrent requests,
  response state, and trusted proxy behavior.
- Runtime: graceful shutdown, readiness/liveness, static-file cache headers, path
  containment, request limits, streaming disconnects, SSR concurrency, and production
  error pages.
- Actions: exact origin policy, CSRF token flow, malformed payload limits, action
  name validation, redirect safety, replay behavior, and rate limiting.
- ORM: real database integration, context cancellation, pool exhaustion, transaction
  rollback, identifier injection, migration locking, partial failure, and concurrent
  migrators.
- Auth/OAuth/RBAC: cookie attributes, proxy model, session fixation, JWT claims and
  key rotation, PKCE/state replay, callback errors, role/permission denial, and
  timing-sensitive paths.
- Component/WASM: hydration mismatch, event listener cleanup, keyed moves, hook
  invariants, safe HTML, browser interaction, and bundle-size checks.
- CLI/generated apps: clean-machine `new`, `dev`, `build`, standalone build, `start`,
  export, database commands, and LSP behavior on Linux/macOS/Windows.
- WebSocket: handshake authorization, origin policy, message limits, backpressure,
  slow clients, room cleanup, close behavior, and race/load tests.

Coverage baseline from this audit:

| Package | Coverage |
| --- | ---: |
| `internal/cli` | 9.6% |
| `internal/compiler` | 54.5% |
| `internal/generator` | 63.1% |
| `internal/lsp` | 58.1% |
| `pkg/action` | 64.5% |
| `pkg/auth` | 37.0% |
| `pkg/auth/oauth` | 51.0% |
| `pkg/component` | 50.4% |
| `pkg/orm` | 32.3% |
| `pkg/rbac` | 48.6% |
| `pkg/router` | 27.6% |
| `pkg/studio` | 40.1% |
| `pkg/ws` | 9.9% |
| `runtime/server` | 49.2% |
| `pkg/env`, `pkg/html`, `internal/livereload`, `internal/watcher` | 0.0% |

The v1.0 release gate should use critical-path contract tests and integration tests,
not a single global percentage target. A provisional minimum is 80% statement
coverage for security-sensitive packages plus 100% acceptance coverage for each
public release promise.

## 7. Documentation and product-contract gaps

Before marketing GoKS as a production framework, documentation must distinguish:

- implemented and tested behavior;
- implemented but experimental behavior;
- planned behavior; and
- examples that are illustrative only.

The following contract decisions are required:

1. Supported Go versions must be one consistent value across `go.mod`, README, CI,
   generated applications, and release notes.
2. Supported operating systems and architectures must be tested or explicitly limited.
3. Database dialects and driver installation must be explicit.
4. The zero-JS fallback must be documented per feature, not as a blanket claim.
5. WASM compiler options, TinyGo support, expected bundle sizes, and limitations must
   be measured rather than advertised as fixed outcomes.
6. Public `pkg/*` APIs need an API inventory, stability labels, deprecation policy,
   examples, and migration notes.
7. Security configuration must include secrets, cookies, trusted proxies, origins,
   CSRF, headers, request limits, and deployment topology.
8. The release process must include checksums, reproducible build inputs, SBOM or an
   explicit v1 deferral, vulnerability scanning, and rollback instructions.

## 8. Recommended execution order

The next implementation batches should be:

### Batch A — P0 security and build correctness

- Exact origin policy and CSRF boundary.
- Safe redirect and trusted-proxy policy.
- Explicit trusted HTML/raw DOM sink policy.
- Build failure behavior for required CSS/assets and dependency preparation.
- Regression tests for all of the above.

### Batch B — P1 runtime and data correctness

- Response writer state machine and request lifecycle.
- Remove request-specific global state and improve shutdown.
- ORM identifier safety, driver matrix, transaction/migration guarantees.
- Session cookie and JWT deployment contract.
- Integration tests against supported databases and concurrent HTTP workloads.

### Batch C — P1 component/WASM correctness

- Safe prop equality and keyed reconciliation invariants.
- Renderer listener lifecycle and browser interaction tests.
- Hook invariants and hydration mismatch behavior.
- Bounded/cancelable streaming SSR.

### Batch D — release and developer experience

- CI matrix and generated-app acceptance fixtures.
- API compatibility checks and documentation synchronization.
- Examples, migration guide, security guide, deployment guide, and troubleshooting.
- Release candidate checklist and v1.0.0 rehearsal.

## 9. Audit decision

Decision: GoKS is suitable for continued engineering toward v1.0.0, but not yet for
production-readiness claims or a stable v1 release.

The next code change must be a security-focused P0 fix with regression tests, starting
with exact origin validation and the state-changing action boundary. Feature expansion
should pause until the P0 items and the critical-path integration-test harness are in
place.

Detailed milestone tasks, dependencies, and acceptance criteria remain in
`docs/task/ROAD_TO_V1.md`.

## 10. Domain audit addendum

The following findings were independently reviewed against the source after the initial audit draft. They are blockers or high-priority follow-up items for the roadmap.

### 10.1 Runtime, router, middleware, and WebSocket

- `HIGH`: `runtime/server/server.go:82-86` installs only Logger and Recover by default, while `.agents/architecture/runtime.md:8-14` describes CORS, Secure, RequestID, Compress, MaxBytes, and Timeout as runtime invariants.
- `HIGH`: `pkg/router/router.go:102-112` resolves routes in registration order. This makes `/*` registered before `/` win because `pkg/router/router.go:143-158` allows the wildcard to match `/`; runtime registrations are visible at `runtime/server/server.go:258-277` and `349-365`.
- `CRITICAL`: `pkg/ws/ws.go:69-75` closed a client channel on backpressure while `pkg/ws/ws.go:113-120` closed the same channel during reader cleanup. `Client.Send` also sent directly to the channel at `pkg/ws/ws.go:79-82`. This has been remediated in the current work with synchronized enqueue/close lifecycle and regression tests.
- `HIGH`: `pkg/ws/ws.go:19-31` previously accepted host-suffix Origins and had no authentication or message limits. Exact origin comparison is now implemented, but authentication, read limits, deadlines, and runtime hub shutdown ownership remain open.
- `HIGH`: `pkg/router/middleware.go:10-25` wraps ResponseWriter without preserving `http.Hijacker` or `http.Flusher`; this can break WebSocket upgrades and streaming when Logger is installed.
- `HIGH`: `pkg/router/mw_limits.go:28-70` runs handlers in a goroutine and mutates the same Context around timeout response handling. This needs a dedicated concurrency test and a redesign that relies on request cancellation rather than unsafe Context mutation.
- `HIGH`: `runtime/server/server.go:412-432` locks the global SSR mutex without a deferred unlock, so a panic during expansion can block later SSR requests.

### 10.2 GOX, component, SSR, and WASM

- `CRITICAL`: `internal/compiler/gox.go:294-300` passes component children to `formatGoNode`, but `internal/compiler/gox.go:349-360` generates component props without using `inner`. This contradicts the nested component syntax documented in `README.md:174-182`.
- `HIGH`: `internal/compiler/gox.go:303-314` treats every end tag as the current node terminator and only recognizes a complete `{expression}` character-data token. Mismatched tags and mixed text interpolation need compiler errors/tests.
- `CRITICAL`: `pkg/component/renderer.go:62-66` clears the SSR root and rebuilds the DOM; it does not hydrate existing markup. Browser integration tests are required before claiming SSR/WASM hydration.
- `HIGH`: `pkg/component/reconciler.go:80-114` computes keyed matches but emits no reorder patch, while `pkg/component/renderer.go:293-296` always appends PatchCreate nodes. Keyed reorder and indexed insertion are not implemented.
- `HIGH`: `pkg/component/reconciler.go:127-147` compares arbitrary interface values with `!=`, which can panic for maps/slices; `pkg/component/node.go:111-115` assumes an existing class value is a string.
- `HIGH`: `pkg/component/ssr.go:33-45` and `pkg/component/stream.go:168-189` emit tag/attribute names and innerHTML without a trusted-HTML boundary. `pkg/component/ssr.go:43-45` also does not reject dangerous URL schemes such as `javascript:`.
- `HIGH`: `pkg/store/store.go:70-80` captures a subscriber slice index in the unsubscribe closure. Out-of-order unsubscribe can panic; the package has no tests.

### 10.3 ORM, auth, OAuth, RBAC, actions, and RPC

- `P0`: the default database CLI path is not operational. `internal/cli/db.go:58-60` calls `orm.OpenSQLite`, but no SQLite driver is registered in `go.mod`; `goks db status` was verified to fail with `unknown driver "sqlite3"`.
- `HIGH`: `pkg/orm/orm.go:95-99` and `159-166` append raw `Where` condition text; the public API needs structured predicates or an explicitly trusted raw SQL escape hatch.
- `HIGH`: `pkg/orm/migrate.go:163-184` can delete migration tracking for a missing migration definition without executing Down; migration DB read/scan errors are also converted into empty state at `43-64`.
- `HIGH`: sessions are process-local at `pkg/auth/auth.go:31-47`, JWT logout is not revocation at `pkg/auth/jwt.go:79-106`, and every manager starts an uncancellable GC goroutine at `pkg/auth/auth.go:164-176`.
- `HIGH`: OAuth PKCE is not mandatory at `pkg/auth/oauth/oauth.go:113-165`; provider identity verification is incomplete at `264-297` and `348-365`.
- `HIGH`: `pkg/rbac/rbac.go:78-118` reads session CurrentUser, while `pkg/auth/mw_jwt.go:57-87` populates JWT claims separately; JWT-authenticated requests do not share the same RBAC path.
- `HIGH`: `pkg/rpc/server.go:52-60` repeats the weak Origin check and bypasses the action validation/auth path described by `ARCHITECTURE.md:65-71`.
- The action Origin comparison and external redirect surface identified at `pkg/action/action.go:95-100` and `149-164` were remediated in the current work with exact origin and same-site redirect tests. CSRF tokens, per-action authorization, and RPC/action unification remain open.

### 10.4 CLI, compiler pipeline, documentation, and release

- `P0`: `internal/cli/export.go:85-92` builds from the project tree but `114-126` runs the exporter from `.goks/entry`; `runtime/server/server.go:722-738` resolves assets relative to AppDir. This requires a real CLI export integration test and likely working-directory correction.
- `HIGH`: `internal/cli/build.go:58-61` and `internal/cli/export.go:89-92` ignore `go mod tidy` errors; CSS download/compile failures are suppressed at `internal/cli/build.go:203-218`.
- `HIGH`: generator path confinement is missing in `internal/cli/new.go:28-30`, `internal/cli/generate.go:34-35` and `107-115`, and `internal/cli/page.go:54-60`; traversal and invalid Go identifier tests are required.
- `HIGH`: no tracked CI workflow was found. Coverage is especially low in `internal/cli` (9.6%), `internal/watcher` (0.0%), `internal/livereload` (0.0%), and `runtime/server` (49.2%).
- `P0` (resolved): `go.mod` now declares Go 1.26.6, and README, scaffold
  templates, generated entry modules, Docker, and CI examples were aligned to
  1.26.6+. The bump also clears the six standard-library advisories reported by
  `govulncheck` against the 1.26.5 toolchain. A cross-platform build matrix
  (linux/arm64, darwin/arm64, windows/amd64) is now enforced in CI.
- `HIGH`: automatic Tailwind/TinyGo downloads in `internal/cli/tailwind.go:30-82` and `internal/cli/tinygo.go:85-151` lack checksum/signature verification.

## 11. Revised status and next implementation slices

M0.1 is complete as an audit deliverable. The first remediation slice is complete and verified in `pkg/action` and `pkg/ws`. The next P0 slices are:

1. Make `goks db status` work in a fresh scaffold by choosing and documenting driver ownership, then add a generated-app smoke test.
2. Fix WebSocket/runtime protocol preservation, authentication boundary, limits, and shutdown ownership.
3. Fix GOX nested children before adding more component syntax.
4. Add the generated-app end-to-end test required by `docs/task/ROAD_TO_V1.md`.
5. Add CI with native/WASM/race/security/version-matrix gates before claiming production readiness.

### Finding ownership map

| Finding group | Owner | Target milestone |
|---|---|---|
| Action/RPC CSRF and authorization | Security/runtime maintainer | M0.4, M3.4 |
| WebSocket lifecycle, origin, limits, and shutdown | Runtime maintainer | M1.2, M1.5 |
| Router precedence and ResponseWriter capabilities | Router/runtime maintainer | M1.1, M1.3 |
| ORM drivers, raw predicates, and migration semantics | Data/ORM maintainer | M3.1, M3.2 |
| Session/JWT/OAuth/RBAC integration | Security/auth maintainer | M3.3, M3.4 |
| GOX parser and nested component generation | Compiler maintainer | M4.1, M4.2 |
| Hydration, keyed reconciliation, SSR sinks, and store | Component/WASM maintainer | M4.3, M4.4 |
| CLI path safety, export, download verification, and Go matrix | Tooling/release maintainer | M5.1, M5.2, M7.1 |
| CI, generated-app acceptance, and coverage gates | Release maintainer | M0.3, M7.2 |
