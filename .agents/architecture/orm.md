# Architecture Note: ORM (`pkg/orm`)

Fluent, type-safe query builder (`orm.Query[T]().Where(...)`), auto
timestamps, soft deletes, migrations, zero-config local dev via
`orm.OpenSQLite(...)`.

Key invariants:
- Query builder must not be vulnerable to SQL injection — all user input
  goes through parameterized queries, never string concatenation.
- Soft-delete and timestamp behavior must be consistent and documented; a
  query must not silently include soft-deleted rows unless explicitly asked.
- Migrations must be forward-safe and reviewed for destructive operations.
- Transactions must be tested under `go test -race` for connection-pool
  correctness.

Additional database adapters beyond SQLite are a P1/roadmap item — do not
add one without a concrete need (see anti-over-engineering checklist).
