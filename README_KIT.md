# GoKS Agent Engineering Kit

This kit is the "engineering constitution + rules + roadmap" package for
running an AI agent (Hermes, Claude, or others) as a maintainer of the GoKS
framework (https://github.com/misbakhul29/goks), instead of a generic
prompt-driven coding assistant.

## How to use

1. Copy everything in this zip into the root of your `goks` repository,
   preserving the folder structure:
   - `AGENTS.md`
   - `GOKS_CONSTITUTION.md`
   - `ARCHITECTURE.md`
   - `ROADMAP.md`
   - `CONTRIBUTING.md`
   - `SECURITY.md`
   - `.agents/rules/*`
   - `.agents/architecture/*`
   - `.agents/roadmap/*`
   - `docs/adr/*`
2. Commit them: `git add . && git commit -m "docs: add agent engineering constitution and rules"`.
3. Give your agent the instruction in `BOOTSTRAP_TASK.md` as its first task
   — an audit-only task, no code changes.
4. Review `GOKS_ARCHITECTURE_AUDIT.md` once produced, and adjust
   `ROADMAP.md` / `.agents/roadmap/*` if needed.
5. From then on, hand the agent roadmap items one at a time, always
   referencing `AGENTS.md`'s quality gate.

## What's inside

| File/Dir | Purpose |
| --- | --- |
| `AGENTS.md` | The main working contract — mandatory reading order, mission, non-negotiables, quality gate, git discipline. |
| `GOKS_CONSTITUTION.md` | Vision, non-negotiables, and priority ordering for trade-offs. |
| `ARCHITECTURE.md` | Current high-level architecture, package map, lifecycles. |
| `ROADMAP.md` | P0–P2 roadmap index. |
| `CONTRIBUTING.md` | Human-contributor-facing version of the workflow. |
| `SECURITY.md` | Security checklist and vulnerability reporting process. |
| `.agents/rules/` | Always-on rules split by concern (workspace, Go, architecture, security, testing, performance, git). |
| `.agents/architecture/` | Per-package architecture notes (runtime, component, wasm, routing, action, orm, cli). |
| `.agents/roadmap/` | Detail + acceptance criteria for each roadmap phase. |
| `docs/adr/` | Architecture Decision Records already implied by the current README/repo (Go-first, WASM islands, GOX syntax, file-system routing, server actions, ORM, state management), plus a template for new ones. |
| `BOOTSTRAP_TASK.md` | The exact first instruction to give the agent (audit-only, no code changes). |

## Notes

- All content here was drafted from the actual repository structure at
  https://github.com/misbakhul29/goks (packages under `pkg/`, `internal/`,
  `runtime/`, and `main.go`) as of the time this kit was generated — but
  treat it as a first draft. The agent's first job (the bootstrap audit) is
  to verify and correct anything here that no longer matches the code, then
  update these files in the same change.
- This kit intentionally does not create subagent role prompts (Architect,
  Backend, WASM, CLI, Security, Performance, QA) as separate files — those
  are workflows you configure inside your agent platform (e.g. Hermes'
  `delegate_task`/subagents) pointing back at this same `AGENTS.md` contract.
  Say the word if you want those role prompts drafted too.
