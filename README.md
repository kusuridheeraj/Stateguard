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
- [Execution Prompt Pack](docs/agents/execution-prompt-pack.md)
- [Windows + WSL2 Setup](docs/setup/windows-wsl2.md)
- [Linux Setup](docs/setup/linux.md)
- [macOS Setup](docs/setup/macos.md)
- [Kubernetes Beta Setup](docs/setup/kubernetes-beta.md)
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
stateguard scheduler
stateguard retention
stateguard compose inspect -f examples/windows-wsl2-compose/compose.yaml
```

Current service entrypoints:

- `cmd/daemon` starts the daemon HTTP scaffold
- `cmd/dashboard-api` starts the dashboard API scaffold
- `cmd/cli` provides the first command surface

## License

Apache License 2.0. See [LICENSE](LICENSE).
