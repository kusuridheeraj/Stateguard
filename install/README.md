# Installation

Stateguard v1 is designed to install:

- a privileged host daemon
- the `stateguard` CLI
- the dashboard API and web console

## Installation Targets

- Windows with Docker Desktop + WSL2
- Linux
- macOS

## Current State

The scripts in this directory are executable installers. They expect prebuilt binaries in `dist/<platform>` and perform the following:

- create install, config, and artifact directories
- copy the CLI, daemon, and dashboard API binaries
- write a default `safedata.yaml` if one does not exist
- register startup services or startup tasks
- create a safe Compose wrapper command that routes through `stateguard intercept compose`

By default, Compose backup execution metadata is generated in planned mode. Set `runtime.compose.live_execution: true` in `safedata.yaml` when you want the daemon to attempt live backup command execution for supported adapters.

Each installer also supports a `validate-only` mode that writes into temporary roots and verifies the produced layout without registering services or touching host startup state. The validation wrappers in `install/validation` are what CI and manual validation should run first.

## Expected Install Layout

- configuration: host-managed `safedata.yaml`
- artifacts: host-managed local artifact directory
- binaries: platform-appropriate install location
- daemon: registered as a startup task, system service, or launch daemon
- Compose wrapper: platform-local helper command that routes destructive intent through Stateguard

## Platform Notes

- Windows installer registers `StateguardDaemon` and `StateguardDashboardAPI` scheduled tasks under `SYSTEM`
- Linux installer writes `systemd` units and enables them immediately
- macOS installer writes `launchd` plists and bootstraps them immediately
