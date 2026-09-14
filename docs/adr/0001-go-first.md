# ADR-0001: Go-First, Zero Required Node.js/npm

Status: Accepted

## Context

GoKS aims to let developers build fullstack web applications using only Go,
avoiding the JS/Node.js toolchain (npm install, node_modules, JS build
steps) that most modern web frameworks require.

## Decision

GoKS will never require Node.js or npm to develop, build, or run a GoKS
application. Tailwind CSS is bundled via a standalone compiler binary, not
npm. Client interactivity is delivered via Go-compiled WebAssembly, not
JavaScript.

## Consequences

Positive:
- Single-language stack lowers cognitive overhead for Go developers.
- No `node_modules`/npm supply-chain surface.
- Single standalone binary deployment story.

Negative:
- Cannot directly reuse the vast JS/npm ecosystem (UI libraries, build
  tools); GoKS must provide its own (`goks ui`, GOX, Tailwind standalone).
- WASM binary size and browser compatibility become GoKS's own
  responsibility (see ADR-0002).
