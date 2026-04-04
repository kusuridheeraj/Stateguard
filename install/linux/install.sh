#!/usr/bin/env sh
set -eu

INSTALL_ROOT="/opt/stateguard"
CONFIG_ROOT="/etc/stateguard"
ARTIFACT_ROOT="/var/lib/stateguard/artifacts"

echo "Preparing Stateguard directories..."
mkdir -p "$INSTALL_ROOT" "$CONFIG_ROOT" "$ARTIFACT_ROOT"

echo "Phase 4 installer scaffold complete."
echo "Next phases will place binaries, register services, and install default config."
