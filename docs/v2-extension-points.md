# V2 Extension Points

This document describes where the current Stateguard architecture is intentionally open for v2 development.

## Remote Artifact Backends

Current state:

- local host-managed artifact storage only

Planned extension points:

- S3-compatible object storage
- Azure Blob Storage
- GCS
- remote retention policies
- multi-tier artifact placement

## Multi-Host Restore

Current state:

- same-host recovery focus

Planned extension points:

- restore artifact portability metadata
- host targeting and restore remapping
- cross-host restore validation workflows

## Kubernetes Enforcement

Current state:

- beta manifest inspection and workload classification

Planned extension points:

- admission control integration
- release-aware protection policy
- delete and uninstall interception
- richer namespace and application boundary modeling

## Adapter Maturity

Current state:

- Postgres, Redis, and Vault have service-aware manifest and validation logic
- MySQL, MongoDB, Kafka, and generic fallback still need equivalent specialization

Planned extension points:

- service-aware backup execution
- service-aware validation
- point-in-time recovery for supported systems
- adapter capability negotiation

## Artifact Efficiency

Current state:

- manifest-backed metadata and local files

Planned extension points:

- deduplication
- incremental artifacts
- compaction
- retention by policy profiles

## CI/CD Integration

Current state:

- baseline CI for repo validation

Planned extension points:

- GitHub Actions integration
- Azure DevOps integration
- Jenkins integration
- deployment-time protection checks
