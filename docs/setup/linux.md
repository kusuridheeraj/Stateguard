# Linux Setup

## Target

- Linux host
- Docker Engine or compatible Compose runtime

## Expected V1 Flow

1. Install Stateguard through the installer or native package.
2. Register the daemon as a system service.
3. Configure artifact storage, retention, and optional overrides in `safedata.yaml`.
4. Validate protected Compose project boundaries.

## Current Phase 4 Scaffolds

- installer scaffold: `install/linux/install.sh`
- Linux package metadata placeholder: `packaging/nfpm.yaml`

## Notes

- Linux Compose support is a stable target in the first public release.
- Same-host storage is the v1 recovery tier.
