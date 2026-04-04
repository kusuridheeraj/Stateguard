# Contributing to Stateguard

Stateguard is intended to be a serious open-source infrastructure project. Contributions should optimize for safety, recoverability, clarity of guarantees, and operational honesty.

## Development Principles

- Do not claim stronger recovery semantics than the code can prove.
- Separate strong guarantees from best-effort fallback in code and docs.
- Prefer explicit policy and diagnostics over silent behavior.
- Keep cross-platform behavior consistent where possible.
- Treat supported adapters as contracts, not demos.

## Repository Conventions

- `cmd/` contains executable entrypoints.
- `internal/` contains product logic that is not part of the public API.
- `pkg/` contains reusable exported packages and the adapter SDK.
- `adapters/` contains service-specific implementations and tests.
- `docs/adr/` contains architectural decision records.
- `examples/` must stay runnable and aligned with the current docs.

## Contribution Flow

1. Open an issue or discussion for large architectural changes.
2. Add or update an ADR for non-trivial design decisions.
3. Keep pull requests scoped to one concern where possible.
4. Add tests for policy, recovery, or adapter behavior.
5. Update docs when user-visible behavior changes.

## Coding Standards

- Write clear code over clever code.
- Favor explicit errors and structured logging.
- Treat safety-related branches as first-class code paths.
- Avoid introducing platform-specific behavior without tests or docs.

## Security

If you believe you found a security issue, follow [SECURITY.md](SECURITY.md).
