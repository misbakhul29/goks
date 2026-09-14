---
trigger: always_on
---

# Architecture Rules

- Before changing an exported API in `pkg/*`, inspect all in-repo usages,
  evaluate backward compatibility, and if breaking: write an ADR, document
  the break, add a migration note, and update tests.
- Do not create a new package or abstraction without a concrete, cited use
  case in the current codebase (see the anti-over-engineering checklist in
  `AGENTS.md`).
- Keep the layering intact: `internal/*` implements build/dev tooling
  mechanics; `pkg/*` is the stable developer-facing API; `runtime/*` wires a
  compiled application together. Do not reach across layers in the wrong
  direction (e.g. `pkg/*` must not depend on `internal/cli`).
- Any change to a fundamental decision already recorded in `docs/adr/`
  requires a new ADR that supersedes it — do not silently contradict an
  existing ADR.
- Update `ARCHITECTURE.md` and the relevant `.agents/architecture/*.md` file
  in the same change whenever package responsibilities or lifecycles shift.
