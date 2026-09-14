# GoKS Benchmark Methodology & Baseline Performance

This document describes the benchmarking methodology and baseline numbers for performance-critical components in GoKS.

## 1. Methodology

All benchmarks are written using standard Go testing primitives (`testing.B`) and can be executed via:

```bash
go test -bench=. -benchmem ./...
```

To ensure benchmark reproducibility:
- Benchmarks avoid external network dependencies.
- In-memory SQLite (`:memory:`) or synthetic mock handlers are used where appropriate.
- Memory allocations (`B/op`) and allocation counts (`allocs/op`) are measured alongside execution latency (`ns/op`).

---

## 2. Benchmark Baselines

### 2.1 Router Matching (`pkg/router`)
Benchmarks route dispatching, parameter extraction, and URL unescaping across static, dynamic, and catch-all routes:
- **Static Route Match:** ~30 ns/op (0 allocs/op)
- **Parameterized Route Match (`/users/:id`):** ~120 ns/op (1-2 allocs/op)
- **Catch-All Wildcard Match (`/files/*path`):** ~140 ns/op (2 allocs/op)

Location: `pkg/router/router_bench_test.go`

### 2.2 Virtual DOM Reconciliation & SSR (`pkg/component`)
Benchmarks synchronous HTML serialization and DOM diffing:
- **RenderToString (Deep Component Tree):** ~2,500 ns/op
- **Tree Reconciliation (No Change):** ~150 ns/op (0 allocs/op)
- **Tree Reconciliation (Prop & Text Update):** ~600 ns/op

Location: `pkg/component/component_bench_test.go`

### 2.3 WebSocket Hub Event Dispatching (`pkg/ws`)
Benchmarks message broadcast to multiple room subscribers under concurrent load:
- **Room Broadcast (100 subscribers):** ~45,000 ns/op
- **Direct Client Send:** ~350 ns/op

Location: `pkg/ws/ws_bench_test.go`
