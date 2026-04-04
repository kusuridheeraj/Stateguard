# Security Policy

Stateguard is intended to operate with elevated host privileges and to manage recovery artifacts for stateful services. Security issues should be treated seriously.

## Reporting

Do not open public issues for sensitive security problems.

Until a dedicated reporting channel is published, contact the maintainers privately through the repository owner profile and mark the report as security-sensitive.

## Scope

Examples of high-priority security issues:

- privilege escalation through the daemon, installer, or runtime integrations
- unauthorized access to recovery artifacts
- leakage of service credentials, secrets, or backup data
- unsafe restore behavior that could expose protected workloads
- bypasses in destructive-command interception or enforcement
