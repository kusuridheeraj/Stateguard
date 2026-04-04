# Stateguard Execution Prompt Pack

This file defines the execution-ready multi-agent build plan for Stateguard.

## Master Prompt

```text
Build Stateguard as an open-source Go monorepo.

Mission:
Create a production-oriented state protection platform that prevents accidental data loss from destructive Docker Compose and Kubernetes operations.

Frozen requirements:
- stable Compose support in first public release for Windows/WSL2, Linux, macOS
- Kubernetes included in first public release as beta
- privileged host daemon as control plane
- explicit safe commands and transparent interception both required
- web dashboard, CLI, and structured logs required in v1
- same-host artifact storage first; remote storage in v2
- Apache-2.0 license
- monorepo
- config file: safedata.yaml

Safety contract:
- protect before destructive actions
- block when no verified recoverable artifact exists
- use background recovery point preparation
- use hybrid validation
- support official strong adapters for Postgres, Redis, Vault, MySQL, MongoDB, Kafka
- use generic fallback for unknown services
- remain explicit about strong guarantees vs best-effort fallback
```

## Agent Roster

### Principal Architect

Ownership:

- `docs/architecture.md`
- `docs/adr/`
- product contract
- guarantees and non-goals

Deliver:

- architecture refinements
- ADR set
- interface boundaries
- risk register

### Core Platform

Ownership:

- `cmd/daemon`
- `internal/policy`
- `internal/orchestrator`
- `internal/artifacts`
- `internal/retention`
- `internal/scheduler`
- `pkg/types`
- `pkg/logging`

Deliver:

- daemon skeleton
- policy engine
- artifact metadata store
- retention engine
- scheduler loop
- health and metrics surfaces

### Compose Runtime

Ownership:

- `internal/compose`
- `internal/intercept`
- `examples/windows-wsl2-compose`
- `examples/linux-compose`
- `examples/macos-compose`

Deliver:

- Compose workload discovery
- project boundary logic
- destructive command routing design
- stable runtime examples

### Adapter Team A

Ownership:

- `adapters/postgres`
- `adapters/mysql`
- `adapters/mongodb`

Deliver:

- adapter contracts
- persistence detection
- protect/validate/restore scaffolds

### Adapter Team B

Ownership:

- `adapters/redis`
- `adapters/vault`
- `adapters/kafka`
- `adapters/generic`

Deliver:

- adapter scaffolds
- generic fallback behavior
- degraded protection policy hooks

### Kubernetes Beta

Ownership:

- `internal/kube`
- `examples/kubernetes-beta`

Deliver:

- beta architecture
- admission/controller integration plan
- examples and warnings

### Dashboard and API

Ownership:

- `cmd/dashboard-api`
- `web/`

Deliver:

- API surface
- dashboard skeleton
- pages for status, recovery points, and warnings

### Installer and Release

Ownership:

- `install/`
- `packaging/`
- `.github/workflows/`

Deliver:

- installer design
- service registration flow
- packaging stubs
- release and CI workflow

### Docs and DX

Ownership:

- `README.md`
- `CONTRIBUTING.md`
- `SECURITY.md`
- setup guides
- examples documentation

Deliver:

- onboarding docs
- contributor docs
- setup and environment guides

### Launch and Marketing

Ownership:

- `blog/`
- project positioning
- homepage copy

Deliver:

- launch blog
- deep-dive blog
- LinkedIn copy
- X/Twitter thread
- Reddit post
- Medium article draft

## Ownership Rules

- Do not overlap file ownership between agents unless integration is explicitly needed.
- Architecture agent owns contracts and may request interface changes before implementation lands.
- Adapter teams must use the shared SDK and not fork contracts.
- Compose and Kubernetes teams own runtime-specific behavior, not the global policy engine.

## Build Order

### Phase 1

- architecture freeze
- monorepo scaffold
- daemon, CLI, dashboard API bootstrap
- README, roadmap, contributing docs

### Phase 2

- policy engine
- artifact store
- retention engine
- scheduler
- Compose discovery and project boundary logic

### Phase 3

- adapter SDK
- official adapter scaffolds
- generic fallback
- example workloads

### Phase 4

- dashboard UI
- installer and native packaging
- CI workflows
- health, logging, metrics

### Phase 5

- Kubernetes beta integration
- launch assets
- v2 extension-point docs

## V2 Development Track

Parallel design tracks to start after v1 architecture is stable:

- remote artifact backends
- multi-host restore workflows
- stronger Kubernetes enforcement
- additional adapters
- incremental and deduplicated artifact storage
- CI/CD platform integrations
- fleet visibility and policy profiles

## Execution-Ready Prompt Pack

Use the prompts below as-is when spawning agents.

### Agent Prompt: Principal Architect

```text
You own architecture and contracts for Stateguard. Work only in docs and ADRs unless a contract file must be added. Refine the architecture, guarantees, non-goals, and interface boundaries for a Go monorepo that protects stateful workloads before destructive Docker Compose and Kubernetes operations. Do not edit runtime code owned by other agents. List every file you changed.
```

### Agent Prompt: Core Platform

```text
You own the Stateguard daemon control plane. Work only in cmd/daemon, internal/policy, internal/orchestrator, internal/artifacts, internal/retention, internal/scheduler, pkg/types, and pkg/logging. Build production-oriented scaffolding for the daemon, policy evaluation, artifact indexing, retention quotas, scheduling, and health surfaces. You are not alone in the codebase; do not revert others' edits. List every file you changed.
```

### Agent Prompt: Compose Runtime

```text
You own Docker Compose runtime integration for Stateguard. Work only in internal/compose, internal/intercept, and the Compose example directories. Implement project boundary detection, destructive operation intent modeling, and stable runtime examples for Windows/WSL2, Linux, and macOS. You are not alone in the codebase; do not revert others' edits. List every file you changed.
```

### Agent Prompt: Adapter Team A

```text
You own the Postgres, MySQL, and MongoDB adapters. Work only in adapters/postgres, adapters/mysql, adapters/mongodb, and any shared adapter test fixtures explicitly needed. Implement adapter scaffolds that detect persistence, protect, validate, and restore through the shared SDK. You are not alone in the codebase; do not revert others' edits. List every file you changed.
```

### Agent Prompt: Adapter Team B

```text
You own the Redis, Vault, Kafka, and generic adapters. Work only in adapters/redis, adapters/vault, adapters/kafka, adapters/generic, and any shared adapter test fixtures explicitly needed. Implement adapter scaffolds and degraded-protection hooks through the shared SDK. You are not alone in the codebase; do not revert others' edits. List every file you changed.
```

### Agent Prompt: Kubernetes Beta

```text
You own Kubernetes beta integration for Stateguard. Work only in internal/kube and examples/kubernetes-beta. Design and scaffold beta-safe admission/controller integration, workload boundary modeling, and beta example manifests. You are not alone in the codebase; do not revert others' edits. List every file you changed.
```

### Agent Prompt: Dashboard and API

```text
You own the dashboard API and web surface for Stateguard. Work only in cmd/dashboard-api and web. Scaffold the API and dashboard structure for protected workloads, recent recovery points, warnings, blocked operations, and restore history. You are not alone in the codebase; do not revert others' edits. List every file you changed.
```

### Agent Prompt: Installer and Release

```text
You own Stateguard installation and release packaging. Work only in install, packaging, and .github/workflows. Create a one-command installer plan, native package stubs, service registration docs or scripts, and a CI workflow skeleton. You are not alone in the codebase; do not revert others' edits. List every file you changed.
```

### Agent Prompt: Docs and DX

```text
You own Stateguard developer experience documentation. Work only in README.md, CONTRIBUTING.md, SECURITY.md, docs, and examples documentation files. Improve setup clarity, contribution flow, supported-environment guidance, and product honesty. You are not alone in the codebase; do not revert others' edits. List every file you changed.
```

### Agent Prompt: Launch and Marketing

```text
You own Stateguard launch assets. Work only in blog and dedicated marketing docs under docs if needed. Produce launch messaging, technical positioning, contributor-facing onboarding content, and platform-specific social post drafts. You are not alone in the codebase; do not revert others' edits. List every file you changed.
```
