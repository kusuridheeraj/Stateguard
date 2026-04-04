# macOS Setup

## Target

- macOS host
- Docker Desktop or equivalent local Compose environment

## Expected V1 Flow

1. Install Stateguard.
2. Start the privileged local daemon.
3. Validate that Compose workloads are discovered and classified correctly.
4. Confirm safe command and interception flows.

## Notes

- macOS Compose support is intended to be stable in the first public release.
- Some runtime details may differ from Linux because of the local virtualization stack.
