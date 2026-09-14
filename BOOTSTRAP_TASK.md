# Bootstrap Task — Give This To Hermes First

Paste this as the very first instruction to Hermes (or any agent) after it
has access to the GoKS repository with this kit copied in.

---

You are now taking ownership of the GoKS repository as its maintainer agent.

DO NOT modify source code yet.

First:
1. Read AGENTS.md.
2. Read all files under .agents/rules/.
3. Read GOKS_CONSTITUTION.md.
4. Read ARCHITECTURE.md and all files under .agents/architecture/.
5. Read all files under docs/adr/.
6. Inspect the entire repository (pkg/, internal/, runtime/, main.go).
7. Understand the current architecture.
8. Identify public APIs (everything exported under pkg/*).
9. Identify architectural boundaries and any violations of them.
10. Identify technical debt.
11. Identify missing tests.
12. Identify performance risks.
13. Identify security risks (use SECURITY.md's checklist).
14. Identify WASM/GOX architecture problems.
15. Identify CLI/tooling problems.
16. Compare the current architecture against GOKS_CONSTITUTION.md and
    ARCHITECTURE.md.
17. Research modern Go web framework architecture where genuinely useful for
    comparison.

Then produce a file named GOKS_ARCHITECTURE_AUDIT.md at the repository root
containing:
- Current architecture
- Strengths
- Weaknesses
- Critical risks
- Technical debt
- API stability risks
- Performance risks
- Security risks
- Testing gaps
- Developer experience gaps
- Scalability risks
- Recommended architecture changes
- Migration strategy for any recommended changes
- A prioritized roadmap (cross-reference against ROADMAP.md and
  .agents/roadmap/*, updating them if the audit reveals they're wrong)

DO NOT modify production code.
DO NOT create speculative features.
DO NOT rewrite the framework.
This is an architecture discovery task only.

---

After this audit is reviewed by the human maintainer, proceed item by item
through ROADMAP.md, always following the full workflow and quality gate
defined in AGENTS.md.
