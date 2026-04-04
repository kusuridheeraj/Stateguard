#!/usr/bin/env sh
set -eu

SOURCE_ROOT="${SOURCE_ROOT:-$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)}"
INSTALL_ROOT="${INSTALL_ROOT:-/Applications/Stateguard}"
CONFIG_ROOT="${CONFIG_ROOT:-/Library/Application Support/Stateguard}"
ARTIFACT_ROOT="${ARTIFACT_ROOT:-$CONFIG_ROOT/artifacts}"
BIN_ROOT="$INSTALL_ROOT/bin"
DIST_ROOT="$SOURCE_ROOT/dist/macos"
DAEMON_SRC="$DIST_ROOT/stateguardd"
CLI_SRC="$DIST_ROOT/stateguard"
API_SRC="$DIST_ROOT/stateguard-dashboard-api"
DAEMON_DST="$BIN_ROOT/stateguardd"
CLI_DST="$BIN_ROOT/stateguard"
API_DST="$BIN_ROOT/stateguard-dashboard-api"
WRAPPER_DST="$BIN_ROOT/stateguard-compose"
CONFIG_PATH="$CONFIG_ROOT/safedata.yaml"
DAEMON_PLIST="/Library/LaunchDaemons/com.stateguard.daemon.plist"
API_PLIST="/Library/LaunchDaemons/com.stateguard.dashboard.plist"

require_binary() {
  if [ ! -f "$1" ]; then
    echo "missing built binary: $1" >&2
    echo "build release binaries into dist/macos before running the installer" >&2
    exit 1
  fi
}

require_binary "$DAEMON_SRC"
require_binary "$CLI_SRC"
require_binary "$API_SRC"

echo "Preparing Stateguard directories..."
mkdir -p "$INSTALL_ROOT" "$BIN_ROOT" "$CONFIG_ROOT" "$ARTIFACT_ROOT"

echo "Copying binaries..."
install -m 0755 "$DAEMON_SRC" "$DAEMON_DST"
install -m 0755 "$CLI_SRC" "$CLI_DST"
install -m 0755 "$API_SRC" "$API_DST"

if [ ! -f "$CONFIG_PATH" ]; then
  cat >"$CONFIG_PATH" <<EOF
version: "1"
project:
  name: stateguard
  environment: production
policy:
  mode: fail-closed
  validation:
    strategy: hybrid
    integrity_check: true
    allow_degraded: false
    restore_test:
      enabled: true
      cadence: periodic
  retention:
    window: 72h
    max_disk_usage_percent: 20
storage:
  local:
    path: $ARTIFACT_ROOT
runtime:
  compose:
    project_boundary: labels+compose_project
    live_execution: false
  kubernetes:
    mode: beta
daemon:
  host: 127.0.0.1
  port: 7010
api:
  host: 127.0.0.1
  port: 7011
EOF
fi

cat >"$WRAPPER_DST" <<EOF
#!/usr/bin/env sh
exec "$CLI_DST" intercept compose "\$@"
EOF
chmod 0755 "$WRAPPER_DST"

cat >"$DAEMON_PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.stateguard.daemon</string>
  <key>ProgramArguments</key>
  <array>
    <string>$DAEMON_DST</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>STATEGUARD_CONFIG</key>
    <string>$CONFIG_PATH</string>
  </dict>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
EOF

cat >"$API_PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.stateguard.dashboard</string>
  <key>ProgramArguments</key>
  <array>
    <string>$API_DST</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>STATEGUARD_CONFIG</key>
    <string>$CONFIG_PATH</string>
  </dict>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
EOF

launchctl bootstrap system "$DAEMON_PLIST" || true
launchctl bootstrap system "$API_PLIST" || true
launchctl enable system/com.stateguard.daemon
launchctl enable system/com.stateguard.dashboard
launchctl kickstart -k system/com.stateguard.daemon
launchctl kickstart -k system/com.stateguard.dashboard

echo "Stateguard installed."
echo "CLI: $CLI_DST"
echo "Compose wrapper: $WRAPPER_DST"
