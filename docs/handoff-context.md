# Stateguard Handoff Context

This document is the continuation context for another language model or engineer picking up the Stateguard repo.

## Product Contract

Stateguard is an open-source state protection platform for destructive infrastructure operations. It exists to prevent accidental data loss from commands such as:

- `docker compose down`
- `docker compose down -v`
- `docker rm`
- `docker rm -v`
- `docker volume rm`
- `docker system prune`
- Kubernetes delete or uninstall flows later

Core product requirements locked so far:

- stable Compose support on Windows/WSL2, Linux, and macOS
- Kubernetes included as beta
- privileged host daemon
- CLI, dashboard API, web console, structured logs
- explicit safe commands and transparent-interception path
- fail-closed policy when verified protection does not exist
- same-host storage first, remote storage later
- monorepo
- Go implementation
- Apache-2.0 license

## Current Architecture

Primary packages:

- `cmd/daemon`: daemon entrypoint
- `cmd/cli`: CLI entrypoint
- `cmd/dashboard-api`: dashboard/backend API entrypoint
- `internal/service`: control plane orchestration
- `internal/orchestrator`: protection and restore orchestration
- `internal/compose`: Compose discovery and runtime execution
- `internal/backupexec`: backup execution planning and optional live execution
- `internal/kube`: Kubernetes beta inspection and delete guard
- `internal/artifacts`: persisted artifact index/store
- `adapters/`: service-specific adapters
- `web/`: dashboard frontend
- `install/`: platform installers

Important runtime files:

- [controlplane.go](C:\PlayStation\tinkering\Stateguard\internal\service\controlplane.go)
- [protect.go](C:\PlayStation\tinkering\Stateguard\internal\orchestrator\protect.go)
- [compose.go](C:\PlayStation\tinkering\Stateguard\internal\backupexec\compose.go)
- [runner.go](C:\PlayStation\tinkering\Stateguard\internal\compose\runner.go)
- [server.go](C:\PlayStation\tinkering\Stateguard\internal\daemon\server.go)
- [server.go](C:\PlayStation\tinkering\Stateguard\internal\dashboardapi\server.go)

## What Is Implemented

Foundation:

- repo scaffold, license, README, architecture, roadmap, setup docs
- config loading through `safedata.yaml`
- policy engine
- artifact store
- scheduler and retention engine
- Compose project discovery
- Kubernetes beta manifest inspection and delete guard

Runtime:

- Compose guard/intercept flow through control plane
- executable Compose runner for `up` and `down`
- daemon API endpoints for protect, restore, guard, intercept, and kube delete guard
- dashboard API endpoints mirroring daemon behavior
- dashboard action panel for daemon-backed actions
- executable installers for Windows/Linux/macOS with startup registration and Compose wrapper generation
- installer validation wrappers under `install/validation/*` that run installers in `validate-only` mode
- GitHub Actions `install-validation.yml` workflow that builds release binaries and exercises installer validation on Windows, Linux, and macOS
- release workflow now runs `goreleaser check` before publishing

Artifacts:

- bundles now contain:
  - `manifest.json`
  - `checksum.sha256`
  - `capture-plan.json`
  - `restore.sh`
  - `restore.ps1`
  - `execution.json`

Adapters:

- official adapters exist for:
  - `Postgres`
  - `Redis`
  - `Vault`
  - `MySQL`
  - `MongoDB`
  - `Kafka`
  - generic fallback

Concrete live-path work now exists for:

- `Postgres`
- `Redis`
- `Vault`
- `MySQL`
- `MongoDB`
- `Kafka`

These adapters now provide:

- richer execution metadata during protect
- stricter restore validation
- persisted-bundle-based restore semantics instead of trivial `ArtifactID != ""`

Restore:

- first restore orchestration path exists:
  - CLI: `stateguard restore artifact -id <artifact-id>`
  - daemon API: `/api/v1/restore/artifact`
  - dashboard API: `/api/v1/daemon/restore/artifact`

## Current Compose Execution Model

`internal/backupexec` currently supports concrete Compose execution plans for:

- `postgres`
- `redis`
- `vault`
- `mysql`
- `mongodb`
- `kafka`

The generic fallback still stays plan-only when no concrete live execution strategy exists.

Live execution toggle:

- config field: `runtime.compose.live_execution`
- default: `false`

Meaning:

- `false`: write execution plans only
- `true`: attempt to run backup commands through Docker Compose for supported adapters

## Current Restore Semantics

Implemented:

- persisted artifact lookup from store
- adapter resolution by stored adapter metadata
- adapter-specific restore logic
- end-to-end CLI protect then restore smoke path verified for Postgres

Important nuance:

- `Postgres` and `Redis` restore now require persisted bundle files
- `MySQL`, `MongoDB`, and `Kafka` restore now require persisted bundle files
- `Vault` restore locates the bundle via configured artifact storage and validates bundle contents

## Current Interception Model

Implemented:

- Compose interception through explicit CLI/API control-plane paths
- raw Docker argument interception for:
  - `docker compose -f <file> down`
  - `docker compose -f <file> down -v`
  - `docker compose -f <file> up`
  - `docker rm`
  - `docker rm -v`
  - `docker volume rm`
  - `docker system prune`

Current limitation:

- raw `docker rm`, `docker volume rm` and `docker system prune` now return structured host-global denials with targets, flags, and warnings preserved
- they are intentionally not auto-executed yet because safe scope mapping is not solved for arbitrary host-level destructive commands

## Current Kubernetes Enforcement Model

Implemented:

- inspect
- protect
- guard-delete
- enforce-delete

Meaning:

- Stateguard now generates protection artifacts for stateful resources found in Kubernetes manifests before evaluating delete safety
- enforcement uses an admission-style review object with explicit protection evidence and required protections
- this is still beta CLI/API enforcement, not controller/admission enforcement inside the cluster

## Testing Status

Passing:

- `go test ./...`
- CLI smoke:
  - `go run ./cmd/cli intercept compose down -f examples/windows-wsl2-compose/compose.yaml`
  - `go run ./cmd/cli kube guard-delete -f examples/kubernetes-beta/manifests.yaml`
  - protect then restore artifact CLI smoke for a generated Postgres artifact

## Known Gaps

This is not yet a production-ready v1. Main gaps:

- live backup execution is only concrete for `Postgres`, `Redis`, `Vault`
- `MySQL`, `MongoDB`, and `Kafka` still need concrete live execution and restore semantics
- transparent interception is still Stateguard-controlled, not a universal Docker socket enforcement layer
- Kubernetes is still beta guard/evaluation, not admission/controller enforcement
- no remote artifact storage yet
- no true PITR orchestration yet
- restore execution is bundle-validated, not full service replay/bring-up across all adapters
- install/package publishing is not finished
- installer validation is now executable and release validation is wired into CI, but real target validation still needs to be run on disposable hosts

## Immediate Next Priorities

1. Add richer end-to-end integration tests for protect/intercept/restore against live services.
2. Harden transparent interception beyond the current controlled CLI/API path.
3. Validate installers and release assets on real Windows/Linux/macOS targets.
4. Push Kubernetes from beta CLI/API enforcement toward controller/admission enforcement.
5. Add remote artifact storage and stronger disaster-recovery flows.

## Important Collaboration Notes

- docs are expected to be updated in the same pass as code changes
- use `apply_patch` for manual edits
- do not revert unrelated user changes
- this repo has already been committed and pushed incrementally many times; preserve that pattern
