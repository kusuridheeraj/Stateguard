# Stateguard Handoff Context

This document is the continuation context for another language model or engineer picking up the Stateguard repo.

## Product Contract

Stateguard is an open-source state protection platform for destructive infrastructure operations. It exists to prevent accidental data loss from commands such as:

- `docker compose down`
- `docker compose down -v`
- `docker rm`
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

Those three now provide:

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

For the rest of the adapters, execution metadata is still written in plan-only mode. This is intentional so unsupported live execution does not break orchestration.

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
- `Vault` restore locates the bundle via configured artifact storage and validates bundle contents

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

## Immediate Next Priorities

1. Extend `internal/backupexec` live execution support to `MySQL`, `MongoDB`, and `Kafka`.
2. Make restore execution more concrete for those same adapters.
3. Add richer end-to-end integration tests for protect/intercept/restore.
4. Harden installer and release packaging into actual publishable assets.
5. Push Kubernetes from beta guard checks toward controller/admission enforcement.

## Important Collaboration Notes

- docs are expected to be updated in the same pass as code changes
- use `apply_patch` for manual edits
- do not revert unrelated user changes
- this repo has already been committed and pushed incrementally many times; preserve that pattern
