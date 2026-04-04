# ADR 0001: Product Scope and Safety Contract

## Status

Accepted

## Context

The project exists to prevent accidental data loss caused by destructive infrastructure operations, especially in environments where stateful services are misconfigured, volumes are missing, or operators assume persistence exists when it does not.

## Decision

Stateguard will ship as a Go monorepo with:

- a privileged host daemon
- CLI
- web dashboard
- stable Docker Compose support on Windows/WSL2, Linux, and macOS
- Kubernetes beta support in the first public release
- same-host artifact storage as the initial recovery tier

The product will enforce fail-closed behavior when no verified recoverable artifact exists.

## Consequences

- the product must distinguish strong guarantees from best-effort fallback
- background protection is required to make command-time safety practical
- service-aware adapters are necessary for serious data stores
- remote disaster recovery is deferred to v2
