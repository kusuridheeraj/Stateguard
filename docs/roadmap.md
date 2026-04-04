# Roadmap

## V1

Primary outcome:

- a trustworthy same-host state protection platform for accidental destructive operations

Shipped capabilities:

- host daemon for Windows/WSL2, Linux, macOS
- CLI and explicit safe commands
- transparent interception path
- web dashboard
- stable Compose support on Windows/WSL2, Linux, macOS
- Kubernetes beta support
- local artifact store with retention and disk quotas
- hybrid validation
- adapter SDK
- strong adapters for Postgres, Redis, Vault, MySQL, MongoDB, Kafka
- generic fallback adapter for unknown stateful services
- adapter registry and orchestration-backed recovery bundles
- dashboard web console scaffold served by the dashboard API
- executable installers for Windows, Linux, and macOS
- installer and native package distribution where practical
- Kubernetes beta guard flows and examples
- launch and contributor content drafts
- v2 extension-point documentation
- examples and onboarding docs

Execution-track progress beyond the named phases:

- service-aware manifest and validation behavior for Postgres, Redis, and Vault
- service-aware manifest and validation behavior for MySQL, MongoDB, and Kafka
- stronger orchestration tests for multi-service Compose protection and artifact bundle creation
- guard plus execution paths for risky Compose operations
- daemon and dashboard API integration-style endpoint tests for protect, guard, intercept, and Kubernetes delete guard
- startup-service installers plus safe Compose wrappers for Windows, Linux, and macOS

## V2

Primary outcome:

- broaden the platform from accidental-deletion recovery to stronger disaster and fleet-scale recovery workflows

Planned capabilities:

- remote artifact backends
- multi-host restore workflows
- richer Kubernetes enforcement
- more adapters
- disaster-recovery runbooks
- policy profiles by environment
- improved runtime interception depth
- deduplication and incremental artifact strategies
- tighter CI/CD integrations
- fleet visibility and policy profiles

## Non-Goals for V1

- universal exact point-in-time recovery for arbitrary unknown services
- host-loss recovery using local-only artifact storage
- pretending large live datasets can be fully backed up and restore-tested in one second
