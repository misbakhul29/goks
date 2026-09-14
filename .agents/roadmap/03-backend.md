# Roadmap Detail: P1 — Data / Backend

## Goal
Make `pkg/orm` and server action data flow trustworthy for production use.

## Items
- Audit `pkg/orm` query builder for injection-safety (parameterization, no
  string-concatenated SQL).
- Document and test soft-delete + timestamp interaction (default query
  scope).
- Transaction and connection-pool tests under `go test -race`.
- Evaluate (do not build speculatively) demand for a second database
  adapter beyond SQLite.

## Acceptance Criteria
- `pkg/orm` has documented transaction semantics and passing race tests.
- No adapter added without a cited concrete use case.
