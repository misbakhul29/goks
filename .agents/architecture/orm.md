# Architecture Note: ORM (`pkg/orm`)

Fluent, type-safe query builder (`orm.Query[T]().Where(...)`), auto
timestamps, soft deletes, lifecycle hooks, migrations, zero-config local dev via
`orm.OpenSQLite(...)`.

Key invariants:
- Query builder must not be vulnerable to SQL injection: all user input
  goes through parameterized queries, never string concatenation.
- Soft-delete and timestamp behavior must be consistent and documented: a
  query must not silently include soft-deleted rows unless explicitly asked.
- Model lifecycle hooks (`BeforeCreate`, `AfterCreate`, `BeforeUpdate`,
  `AfterUpdate`, `BeforeDelete`, `AfterDelete`, `AfterFind`) are supported
  with either context-aware or plain signatures. Errors returned from Before*
  hooks immediately abort the database operation and roll back pending transactions.
- Reflection logic for IDs (`extractModelID`, `setModelID`) and timestamps
  must handle both embedded `orm.Model` and standalone model structs with
  top-level `ID` and `DeletedAt` fields.
- Struct fields tagged with `db:"-"` must never be mapped to database columns.
- Custom table names can be provided via the `TableNamer` interface
  (`TableName() string`).
- Context propagation (`WithContext`, `CreateContext`, `FindByIDContext`,
  `TransactionContext`) must ensure cancellation signals abort queries immediately
  in the underlying SQL driver.
- Migrations must be forward-safe and reviewed for destructive operations.
- Transactions must be tested under `go test -race` for connection-pool
  correctness.
