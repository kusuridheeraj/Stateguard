# Stateguard Architecture

## Summary

Stateguard is a host-level control plane for state protection around destructive infrastructure operations. Its job is not just to create backups, but to decide whether a risky operation may proceed, based on whether a recent recovery point exists and meets policy.

## Design Goals

- protect state before destructive actions
- keep command-time latency low by preparing recovery points continuously
- support stable Compose flows on Windows/WSL2, Linux, and macOS
- include Kubernetes as beta in the first public release
- make guarantees explicit and testable
- keep same-host accidental-deletion recovery as the first shipping recovery tier

## Core Components

### Host Daemon

Responsibilities:

- policy enforcement
- background scheduling
- artifact metadata management
- retention and disk quota management
- runtime discovery
- logging and metrics
- local API for CLI and dashboard

### CLI

Provides:

- safe equivalents for destructive actions
- status and diagnostics
- restore controls
- setup and bootstrap commands

### Transparent Interception

Transparent interception is required for the product vision, but implementation differs by runtime:

- Compose and Docker flows use a controlled interception path that routes destructive intent through the daemon.
- Kubernetes beta uses admission/controller-style integration and release-aware policy checks.

The current runtime path now includes:

- guard evaluation for risky Compose operations
- control-plane execution methods that run `docker compose down` and `docker compose up` after policy checks
- daemon and dashboard API endpoints that expose the same protect, guard, and intercept behavior
- installer-created wrapper commands that route Compose intent through Stateguard
- optional live backup execution for Compose workloads when `runtime.compose.live_execution` is enabled
- raw Docker argument interception for Compose-scoped destructive intent and policy-aware evaluation for `docker volume rm` and `docker system prune`

### Dashboard

The dashboard should show:

- protected application boundaries
- recent recovery points
- validation state
- blocked operations
- active warnings
- retention pressure and disk usage

The current Phase 4 implementation serves a lightweight web console directly from the dashboard API process so operators can inspect state without a separate frontend build system.

### Adapter Layer

Service-aware adapters provide strong semantics for supported systems:

- Postgres
- Redis
- Vault
- MySQL
- MongoDB
- Kafka

The adapter registry is now a first-class runtime component. It resolves services by runtime hints such as image names, mount characteristics, and stateful heuristics. When no official adapter matches, the generic fallback adapter may still protect the workload with degraded guarantees.

Postgres, Redis, Vault, MySQL, MongoDB, and Kafka now have concrete service-aware manifest and validation behavior beyond the generic scaffold path. The generic fallback remains intentionally less strict and more degraded.

### Orchestration Layer

The orchestration layer turns adapter-level protection results into persisted artifact manifests and metadata records. In the current phase it provides:

- adapter resolution through the registry
- Compose workload to adapter target mapping
- artifact bundle persistence on the local host
- validation result propagation into artifact metadata

Each persisted recovery bundle now contains:

- `manifest.json`
- `checksum.sha256`
- `capture-plan.json`
- `restore.sh`
- `restore.ps1`
- `execution.json`

For `Postgres`, `Redis`, and `Vault`, the orchestration layer now also emits concrete execution metadata that can be used for live backup command execution or restore validation.

`MySQL`, `MongoDB`, and `Kafka` now also emit concrete execution and persisted restore metadata, so the entire initial official adapter set has moved beyond manifest-only restore semantics.

## Protection Lifecycle

1. Discover workloads and infer application boundaries.
2. Detect stateful services using image patterns, labels, runtime signals, and `safedata.yaml`.
3. Prepare recovery points in the background.
4. Validate artifacts using the hybrid model.
5. Intercept or receive a destructive action request.
6. Decide whether a sufficiently recent verified recovery point exists.
7. Allow or block the operation.

In the current implementation phase, the orchestration path is able to simulate protection against Compose workloads and emit persisted artifact bundles and records for supported adapters and generic fallback.

The current execution track now includes concrete manifest generation and validation logic for the full initial official adapter set, with orchestrator tests that verify multi-service Compose flows.

The first restore orchestration path is now wired through artifact lookup, adapter selection, and adapter-specific restore validation using the persisted bundle metadata.

Kubernetes support now includes a protect-before-delete enforcement path that generates artifacts for detected stateful resources in a manifest before evaluating delete safety.

## Hybrid Validation

### Fast Validation

- artifact creation success
- checksum or integrity validation
- metadata and manifest sanity
- adapter-level light checks

### Strong Validation

- isolated restore into a temporary environment
- service boot
- adapter-level health verification

Strong validation is periodic or policy-triggered, not necessarily run for every single destructive command.

## Storage Model

v1 stores artifacts on the same machine in a daemon-managed host directory.

This protects against accidental operator actions such as:

- `docker compose down`
- `docker compose down -v`
- `docker rm`
- `docker volume rm`
- `docker system prune`

It does not replace remote disaster recovery for total host loss. Remote targets are a v2 capability.

## Safety Model

Default enforcement mode is fail-closed:

- if verified protection exists, continue
- if only degraded protection exists, follow explicit policy
- if no valid recoverable artifact exists, block

Container-local storage is never treated as equivalent to durable volumes, even when emergency export is possible.

## Dashboard API Surface

The dashboard API currently exposes operator-facing endpoints for:

- status
- overview
- adapters
- artifacts
- scheduler state
- retention preview
- daemon status
- artifact restore
- Compose protect, guard, and intercept actions
- Kubernetes beta delete guard

The daemon API now exposes protect, restore, guard, and intercept endpoints for risky Compose operations and a beta guard path for Kubernetes delete flows.
