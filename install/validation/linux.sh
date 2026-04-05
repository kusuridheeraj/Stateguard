#!/usr/bin/env sh
set -eu

SOURCE_ROOT="${SOURCE_ROOT:-$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)}"
VALIDATION_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/stateguard-install-validation.XXXXXX")"
INSTALL_ROOT="$VALIDATION_ROOT/install"
CONFIG_ROOT="$VALIDATION_ROOT/config"

cleanup() {
  rm -rf "$VALIDATION_ROOT"
}
trap cleanup EXIT

echo "Running Linux installer validation..."
SOURCE_ROOT="$SOURCE_ROOT" INSTALL_ROOT="$INSTALL_ROOT" CONFIG_ROOT="$CONFIG_ROOT" VALIDATE_ONLY=1 \
  sh "$SOURCE_ROOT/install/linux/install.sh"

BIN_ROOT="$INSTALL_ROOT/bin"
CONFIG_PATH="$CONFIG_ROOT/safedata.yaml"
WRAPPER_PATH="$BIN_ROOT/stateguard-compose"

for path in \
  "$BIN_ROOT/stateguardd" \
  "$BIN_ROOT/stateguard" \
  "$BIN_ROOT/stateguard-dashboard-api" \
  "$CONFIG_PATH" \
  "$WRAPPER_PATH"
do
  if [ ! -f "$path" ]; then
    echo "validation failed: missing $path" >&2
    exit 1
  fi
done

grep -q 'policy:' "$CONFIG_PATH"
grep -q 'validation:' "$CONFIG_PATH"
grep -q 'runtime:' "$CONFIG_PATH"
grep -q 'project_boundary: labels+compose_project' "$CONFIG_PATH"

echo "Linux installer validation passed."
