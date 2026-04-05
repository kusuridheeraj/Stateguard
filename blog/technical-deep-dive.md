# Technical Deep Dive Draft

Stateguard is structured around one core idea: destructive operations should not be treated as normal until state protection has been verified.

## Current architecture

- privileged host daemon
- CLI and dashboard API
- lightweight web console
- policy engine
- artifact metadata store
- scheduler and retention planning
- adapter registry
- orchestration layer for protection flows

## Current runtime coverage

- Docker Compose inspection and protection orchestration
- Raw Docker argument interception for `down`, `rm`, `volume rm`, and `system prune`
- Kubernetes manifest inspection and admission-style delete enforcement

## Why the architecture is layered this way

- daemon owns policy and lifecycle
- adapters own workload-specific behavior
- orchestration turns workload analysis into persisted recovery artifacts
- interception and enforcement layers provide the active safety barrier
- dashboard makes state visible to operators instead of leaving it implicit

## What comes next

- universal transparent Docker socket interception
- in-cluster Kubernetes admission controller
- remote artifact backends
- true PITR orchestration
