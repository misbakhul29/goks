# Contributing to GoKS

Thank you for your interest in contributing to **GoKS**!

GoKS is a production-grade, open-source fullstack web framework built entirely in Go.

## 1. Principles & Non-Negotiables

Before submitting PRs, please review our core engineering principles:

- **Go First:** No Node.js or npm runtime dependency may be introduced. Tailwind CSS is managed via standalone binaries.
- **Zero-JS Interactivity:** Client interactivity features must retain usable fallback functionality when JavaScript/WASM is disabled or unavailable.
- **Public API Stability:** Exported identifiers in `pkg/*` represent public contracts. Breaking changes require an ADR and migration guide.
- **Test Coverage:** Every bug fix or feature must include unit/regression tests. Concurrency-sensitive components must pass `go test -race`.

---

## 2. Platform & Toolchain Support Matrix

- **Go Version:** Go 1.26.6 or higher.
- **Operating Systems:** Linux (amd64, arm64), macOS (Apple Silicon, Intel), Windows (amd64).
- **Client Browsers:** Modern browsers supporting WebAssembly (Chrome 85+, Firefox 78+, Safari 14+, Edge 85+).

---

## 3. Development Workflow & Quality Gate

Every change must pass the repository Quality Gate before merging:

```bash
# 1. Format code
gofmt -l .

# 2. Run static analysis
go vet ./...

# 3. Run unit tests with race detection
go test -race -count=1 ./...

# 4. Verify native build
go build ./...

# 5. Verify WebAssembly compilation
GOOS=js GOARCH=wasm go build ./...
```

---

## 4. Git & Commit Guidelines

We adhere to **Conventional Commits**:
- `feat(scope): ...` — New features.
- `fix(scope): ...` — Bug fixes.
- `refactor(scope): ...` — Code refactoring without behavioral changes.
- `perf(scope): ...` — Performance optimizations.
- `test(scope): ...` — Test additions or improvements.
- `docs(scope): ...` — Documentation updates.

---

## 5. Security & Vulnerability Reporting

Please do not report security vulnerabilities via public GitHub issues. Follow the coordinated disclosure guidelines defined in [SECURITY.md](SECURITY.md).
