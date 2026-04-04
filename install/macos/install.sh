#!/usr/bin/env sh
set -eu

INSTALL_ROOT="/Applications/Stateguard"
CONFIG_ROOT="/Library/Application Support/Stateguard"
ARTIFACT_ROOT="$CONFIG_ROOT/artifacts"

echo "Preparing Stateguard directories..."
mkdir -p "$INSTALL_ROOT" "$CONFIG_ROOT" "$ARTIFACT_ROOT"

echo "Phase 4 installer scaffold complete."
echo "Next phases will place binaries, register launch agents, and install default config."
