---
trigger: always_on
---

# Performance Rules

Every performance-sensitive change should consider: allocations, GC
pressure, memory usage, concurrency/lock contention, I/O, WASM bundle size,
startup time, throughput, and latency.

- Do not claim a performance improvement without a benchmark showing it
  (before/after `go test -bench` numbers, or WASM bundle size before/after).
- For WASM: track bundle size deltas; the standing goal is to keep static
  pages at 0 KB WASM and interactive islands as small as practical (see the
  `--compiler=tinygo` path for size-critical builds).
- Avoid introducing allocations in hot paths (routing match, ORM query
  building, component render/reconciliation) without justification.
