# Installation

Stateguard v1 is designed to install:

- a privileged host daemon
- the `stateguard` CLI
- the dashboard API and web console

## Installation Targets

- Windows with Docker Desktop + WSL2
- Linux
- macOS

## Current Phase 4 State

The scripts in this directory are installer scaffolds. They document the intended installation contract and create the directory layout that later phases will use for service registration and binary placement.

## Expected Install Layout

- configuration: host-managed `safedata.yaml`
- artifacts: host-managed local artifact directory
- binaries: platform-appropriate install location
- daemon: registered as a system service or launch agent
