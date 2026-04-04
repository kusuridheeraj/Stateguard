# Launch Announcement Draft

Stateguard is an open-source state protection platform built to prevent accidental data loss from destructive infrastructure operations.

It exists for the very common failure mode where teams rebuild containers confidently, but stateful services like Postgres, Redis, Vault, Kafka, MongoDB, or MySQL are not actually protected the way everyone assumed they were.

## What the first public release includes

- stable Docker Compose support for Windows/WSL2, Linux, and macOS
- Kubernetes support as beta
- host daemon, CLI, dashboard API, and web console
- adapter registry for core stateful services
- generic fallback handling for unknown stateful workloads
- same-host recovery artifact management

## What it does not pretend to solve yet

- universal exact point-in-time recovery for every workload
- full disaster recovery from host loss using only local storage
- perfect protection for arbitrarily misconfigured systems

The project is being built in public with a strong emphasis on explicit guarantees, contributor friendliness, and operator clarity.
