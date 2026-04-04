# Windows + WSL2 Setup

## Target

- Windows host
- Docker Desktop
- WSL2 backend

## Expected V1 Flow

1. Install Stateguard using the one-command installer or package manager.
2. Register and start the privileged host daemon.
3. Connect the CLI and dashboard to the local daemon.
4. Run Compose workloads through Stateguard-safe flows or configured transparent interception.
5. Confirm protected workloads and recovery points in the dashboard.

## Current State

- executable installer: `install/windows/install.ps1`
- startup tasks for the daemon and dashboard API
- local dashboard API and web console served by the dashboard process
- safe Compose wrapper generation through the installer

## Notes

- This is the first-class stable environment for early validation.
- Recovery artifacts are stored on the host, not inside containers.
