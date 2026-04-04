# Stateguard

Stateguard is an open-source state protection platform for destructive infrastructure operations.

It protects stateful workloads before commands like `docker compose down`, `docker compose down -v`, `docker rm`, `docker volume rm`, `docker system prune`, and later Kubernetes delete or uninstall flows.

## Why Stateguard Exists

Teams lose data for boring reasons:

- a compose file forgot to declare a volume
- someone assumed persistence was already configured
- a destructive command was run during a deploy or local rebuild
- backups existed, but nobody knew whether they were recent, valid, or restorable
- state was stored in container-local writable layers without anyone noticing

Stateguard is designed to close that gap. It does not treat "a backup probably exists somewhere" as protection.

## V1 Scope

Stable in the first public release:

- Docker Compose on Windows with Docker Desktop + WSL2
- Docker Compose on Linux
- Docker Compose on macOS

Included in the first public release:

- Kubernetes support as beta

Included interfaces:

- privileged host daemon
- CLI
- web dashboard
- dashboard API
- structured logs
- explicit safe commands
- transparent interception

Supported strong-protection adapters in v1:

- Postgres
- Redis
- Vault
- MySQL
- MongoDB
- Kafka

## How It Works

1. The host daemon discovers protected application boundaries.
2. A background scheduler prepares recent recovery points continuously.
3. Destructive commands are intercepted or routed through safe commands.
4. The policy engine decides whether a recent verified recovery point exists.
5. If protection is valid, the command continues.
6. If protection is missing or unverifiable, the command is blocked.
7. Supported adapters and the generic fallback emit recovery bundles through the orchestration layer.

Validation is hybrid:

- frequent validation: artifact creation, checksums, metadata sanity checks
- stronger validation on schedule or policy triggers: isolated restore tests for supported services

## Repository Layout

```text
cmd/                 entrypoints for daemon, CLI, and dashboard API
internal/            core orchestration, policy, scheduling, retention, runtime integrations
pkg/                 public SDK and shared packages
adapters/            service-aware recovery adapters plus generic fallback
web/                 dashboard frontend
install/             installer logic and service setup
packaging/           native distribution assets
deploy/              deployment manifests and service definitions
examples/            runnable examples for supported environments
docs/                architecture, ADRs, prompts, contributor docs
blog/                launch and long-form content
test/                integration and recovery validation harnesses
```

## Guarantees and Limits

Strong protection in v1 applies to officially supported adapters when their persistence and recovery settings are correctly configured.

Unknown or weakly configured workloads may still receive generic or emergency protection, but that is not treated as equal to service-aware recovery. Same-host artifact storage protects against accidental destructive operations. It does not replace remote disaster recovery for host failure.

## Key Docs

- [Architecture](docs/architecture.md)
- [Roadmap](docs/roadmap.md)
- [Handoff Context](docs/handoff-context.md)
- [Execution Prompt Pack](docs/agents/execution-prompt-pack.md)
- [Content Prompt Pack](docs/agents/content-prompt-pack.md)
- [Windows + WSL2 Setup](docs/setup/windows-wsl2.md)
- [Linux Setup](docs/setup/linux.md)
- [macOS Setup](docs/setup/macos.md)
- [Kubernetes Beta Setup](docs/setup/kubernetes-beta.md)
- [Installation](install/README.md)
- [Packaging](packaging/README.md)
- [V2 Extension Points](docs/v2-extension-points.md)
- [Contributing](CONTRIBUTING.md)

## Phase 1 Commands

Current scaffolded commands:

```bash
stateguard version
stateguard config validate -f safedata.yaml
stateguard config print-defaults
stateguard status
stateguard policy check
stateguard artifacts
stateguard adapters
stateguard scheduler
stateguard retention
stateguard compose inspect -f examples/windows-wsl2-compose/compose.yaml
stateguard kube inspect -f examples/kubernetes-beta/manifests.yaml
stateguard protect compose -f examples/windows-wsl2-compose/compose.yaml
stateguard restore artifact -id <artifact-id>
stateguard guard compose -f examples/windows-wsl2-compose/compose.yaml --command compose.down
stateguard intercept compose down -f examples/windows-wsl2-compose/compose.yaml
stateguard compose down -f examples/windows-wsl2-compose/compose.yaml --with-volumes
stateguard kube guard-delete -f examples/kubernetes-beta/manifests.yaml
```

Current service entrypoints:

- `cmd/daemon` starts the daemon HTTP scaffold
- `cmd/dashboard-api` starts the dashboard API scaffold
- `cmd/cli` provides the first command surface

Current Phase 3 capabilities:

- official adapter registry for Postgres, Redis, Vault, MySQL, MongoDB, and Kafka
- generic fallback adapter for unknown stateful services
- manifest-backed artifact generation through the orchestration layer
- Compose project inspection and protection simulation against example workloads

Current Phase 4 capabilities:

- dashboard web console served by the dashboard API
- operator API endpoints for status, overview, adapters, artifacts, scheduler state, and retention preview
- executable installers for Windows, Linux, and macOS that copy built binaries, write config, and register host startup services
- package metadata for winget, Homebrew, and Linux package flows

Current Phase 5 capabilities:

- Kubernetes beta manifest inspection
- first-release launch and contributor content drafts
- v2 extension-point documentation

Current execution-track improvements:

- concrete service-aware manifest generation for Postgres, Redis, and Vault
- adapter-specific validation behavior for those services
- stronger orchestration tests covering multi-service Compose protection flows

Current adapter-wave improvements:

- concrete service-aware manifest and validation behavior for MySQL, MongoDB, and Kafka
- orchestration coverage for the full initial official adapter set

Current enforcement-track improvements:

- first guard/evaluation layer for risky Compose operations
- daemon endpoint coverage for guard preflight and status surfaces
- control-plane tests that exercise protection before operation allowance

Current runtime-hardening improvements:

- real control-plane interception methods for `compose down` and `compose up`
- daemon and dashboard API endpoints for protect, guard, intercept, and Kubernetes delete guard paths
- dashboard action panel wired to the daemon-backed endpoints
- Kubernetes delete guard beyond manifest inspection-only CLI output
- recovery bundles written as directories with `manifest.json`, `checksum.sha256`, `capture-plan.json`, `restore.sh`, and `restore.ps1`
- install scripts now register startup tasks/services and create a safe Compose wrapper command

Current live-execution and restore-track improvements:

- Compose protection can now write execution metadata for all supported services and concrete backup command plans for `Postgres`, `Redis`, and `Vault`
- optional live Compose backup execution is configurable via `runtime.compose.live_execution`
- first artifact restore orchestration path is available through the CLI, daemon API, and dashboard API
- `Postgres`, `Redis`, and `Vault` now enforce stricter restore validation semantics instead of accepting arbitrary artifact ids

Dashboard and API surfaces:

- `/`
- `/healthz`
- `/api/v1/status`
- `/api/v1/overview`
- `/api/v1/adapters`
- `/api/v1/artifacts`
- `/api/v1/scheduler`
- `/api/v1/retention/preview`
- `/api/v1/guard/compose`
- `/api/v1/protect/compose`
- `/api/v1/restore/artifact`
- `/api/v1/intercept/compose`
- `/api/v1/guard/kube-delete`
- `/api/v1/daemon/status`
- `/api/v1/daemon/protect/compose`
- `/api/v1/daemon/restore/artifact`
- `/api/v1/daemon/guard/compose`
- `/api/v1/daemon/intercept/compose`
- `/api/v1/daemon/guard/kube-delete`

## License

Apache License 2.0. See [LICENSE](LICENSE).
